/*
 * Copyright 2018-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tomcat

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/jvm-application-cnb/jvmapplication"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/buildpack"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/tomcat-cnb/internal"
)

const (
	// TomcatDependency indicates that Tomcat is required for the web application.
	TomcatDependency = "tomcat"

	// TomcatAccessLoggingSupportDependency is the id for the Tomcat Access Logging Support contributed to the Tomcat instance.
	TomcatAccessLoggingSupportDependency = "tomcat-access-logging-support"

	// TomcatLifecycleSupportDependency is the id for the Tomcat Lifecycle Support contributed to the Tomcat instance.
	TomcatLifecycleSupportDependency = "tomcat-lifecycle-support"

	// TomcatLoggingSupportDependency is the id for the Tomcat Logging Support contributed to the Tomcat instance.
	TomcatLoggingSupportDependency = "tomcat-logging-support"
)

// Tomcat represents a Tomcat instance.
type Tomcat struct {
	application application.Application
	buildpack   buildpack.Buildpack
	layer       layers.MultiDependencyLayer
	layers      layers.Layers
}

// Contribute makes the contribution to launch.
func (t Tomcat) Contribute() error {
	if err := t.layer.Contribute(map[string]layers.MultiDependencyLayerContributor{
		TomcatDependency:                     t.contributeTomcat,
		TomcatAccessLoggingSupportDependency: t.contributeTomcatAccessLoggingSupport,
		TomcatLifecycleSupportDependency:     t.contributeTomcatLifecycleSupport,
		TomcatLoggingSupportDependency:       t.contributeTomcatLoggingSupport,
	}, layers.Launch); err != nil {
		return err
	}

	command := "catalina.sh run"

	return t.layers.WriteApplicationMetadata(layers.Metadata{
		Processes: layers.Processes{
			{"task", command},
			{"tomcat", command},
			{"web", command},
		},
	})
}

func (Tomcat) contextPath() string {
	cp, ok := os.LookupEnv("BP_TOMCAT_CONTEXT_PATH")
	if !ok {
		cp = "ROOT"
	}

	cp = regexp.MustCompile("^/").ReplaceAllString(cp, "")
	return strings.ReplaceAll(cp, "/", "#")
}

func (t Tomcat) contributeTomcat(artifact string, layer layers.MultiDependencyLayer) error {
	layer.Logger.SubsequentLine("Extracting to %s", layer.Root)

	if err := helper.ExtractTarGz(artifact, layer.Root, 1); err != nil {
		return err
	}

	if err := os.RemoveAll(filepath.Join(layer.Root, "webapps")); err != nil {
		return err
	}

	layer.Logger.SubsequentLine("Copying context.xml to %s/conf", layer.Root)
	if err := helper.CopyFile(filepath.Join(t.buildpack.Root, "context.xml"), filepath.Join(layer.Root, "conf", "context.xml")); err != nil {
		return err
	}

	layer.Logger.SubsequentLine("Copying server.xml to %s/conf", layer.Root)
	if err := helper.CopyFile(filepath.Join(t.buildpack.Root, "server.xml"), filepath.Join(layer.Root, "conf", "server.xml")); err != nil {
		return err
	}

	cp := filepath.Join(layer.Root, "webapps", t.contextPath())
	layer.Logger.SubsequentLine("Mounting application at %s", cp)

	return helper.WriteSymlink(t.application.Root, cp)
}

func (t Tomcat) contributeTomcatAccessLoggingSupport(artifact string, layer layers.MultiDependencyLayer) error {
	destination := filepath.Join(layer.Root, "lib", filepath.Base(artifact))

	layer.Logger.SubsequentLine("Copying %s to %s/lib", filepath.Base(artifact), layer.Root)
	if err := helper.CopyFile(artifact, destination); err != nil {
		return err
	}

	return layer.WriteProfile("access-logging", `ENABLED=${BPL_TOMCAT_ACCESS_LOGGING:=n}

if [[ "${ENABLED}" = "n" ]]; then
	return
fi

printf "Tomcat Access Logging enabled\n"

export JAVA_OPTS="${JAVA_OPTS} -Daccess.logging.enabled=enabled"
`)
}

func (t Tomcat) contributeTomcatLifecycleSupport(artifact string, layer layers.MultiDependencyLayer) error {
	destination := filepath.Join(layer.Root, "lib", filepath.Base(artifact))

	layer.Logger.SubsequentLine("Copying %s to %s/lib", filepath.Base(artifact), layer.Root)
	return helper.CopyFile(artifact, destination)
}

func (t Tomcat) contributeTomcatLoggingSupport(artifact string, layer layers.MultiDependencyLayer) error {
	destination := filepath.Join(layer.Root, "bin", filepath.Base(artifact))

	layer.Logger.SubsequentLine("Copying %s to %s/bin", filepath.Base(artifact), layer.Root)
	if err := helper.CopyFile(artifact, destination); err != nil {
		return err
	}

	layer.Logger.SubsequentLine("Copying logging.properties to %s/conf", layer.Root)
	if err := helper.CopyFile(filepath.Join(t.buildpack.Root, "logging.properties"), filepath.Join(layer.Root, "conf", "logging.properties")); err != nil {
		return err
	}

	layer.Logger.SubsequentLine("Writing %s/bin/setenv.sh", layer.Root)
	return helper.WriteFile(filepath.Join(layer.Root, "bin", "setenv.sh"), 0755, `#!/bin/sh

CLASSPATH=$CLASSPATH:%s`, destination)
}

// NewTomcat creates a new Tomcat instance.  OK is true if the application contains a "jvm-application" dependency and a
// "WEB-INF" directory.
func NewTomcat(build build.Build) (Tomcat, bool, error) {
	if _, ok := build.BuildPlan[jvmapplication.Dependency]; !ok {
		return Tomcat{}, false, nil
	}

	ok, err := helper.FileExists(filepath.Join(build.Application.Root, "WEB-INF"))
	if err != nil {
		return Tomcat{}, false, err
	}

	if !ok {
		return Tomcat{}, false, nil
	}

	deps, err := build.Buildpack.Dependencies()
	if err != nil {
		return Tomcat{}, false, err
	}

	tomcatDep, err := tomcatDep(deps, build)
	if err != nil {
		return Tomcat{}, false, err
	}

	accessLoggingDep, err := deps.Best(TomcatAccessLoggingSupportDependency, "", build.Stack)
	if err != nil {
		return Tomcat{}, false, err
	}

	lifecycleDep, err := deps.Best(TomcatLifecycleSupportDependency, "", build.Stack)
	if err != nil {
		return Tomcat{}, false, err
	}

	loggingDep, err := deps.Best(TomcatLoggingSupportDependency, "", build.Stack)
	if err != nil {
		return Tomcat{}, false, err
	}

	return Tomcat{
		build.Application,
		build.Buildpack,
		build.Layers.MultiDependencyLayer("tomcat", []buildpack.Dependency{
			tomcatDep,
			accessLoggingDep,
			lifecycleDep,
			loggingDep,
		}),
		build.Layers,
	}, true, nil
}

func tomcatDep(dependencies buildpack.Dependencies, build build.Build) (buildpack.Dependency, error) {
	version, err := internal.Version(TomcatDependency, build.BuildPlan[TomcatDependency], build.Buildpack)
	if err != nil {
		return buildpack.Dependency{}, err
	}

	return dependencies.Best(TomcatDependency, version, build.Stack)
}

/*
 * Copyright 2019-2020 the original author or authors.
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

package base_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/cloudfoundry/libcfbuildpack/buildpack"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/tomcat-cnb/base"
	"github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestBase(t *testing.T) {
	spec.Run(t, "Base", func(t *testing.T, when spec.G, it spec.S) {

		g := gomega.NewWithT(t)

		var f *test.BuildFactory

		it.Before(func() {
			f = test.NewBuildFactory(t)
		})

		it("returns false with no WEB-INF", func() {
			_, ok, err := base.NewBase(f.Build)
			g.Expect(err).NotTo(gomega.HaveOccurred())
			g.Expect(ok).To(gomega.BeFalse())
		})

		when("valid application", func() {

			it.Before(func() {
				f.AddDependency("tomcat-access-logging-support", filepath.Join("testdata", "stub-tomcat-access-logging-support.jar"))
				f.AddDependency("tomcat-lifecycle-support", filepath.Join("testdata", "stub-tomcat-lifecycle-support.jar"))
				f.AddDependency("tomcat-logging-support", filepath.Join("testdata", "stub-tomcat-logging-support.jar"))
				test.TouchFile(t, filepath.Join(f.Build.Buildpack.Root, "context.xml"))
				test.TouchFile(t, filepath.Join(f.Build.Buildpack.Root, "logging.properties"))
				test.TouchFile(t, filepath.Join(f.Build.Buildpack.Root, "server.xml"))
				test.TouchFile(t, filepath.Join(f.Build.Buildpack.Root, "web.xml"))

				if err := os.MkdirAll(filepath.Join(f.Build.Application.Root, "WEB-INF"), 0755); err != nil {
					t.Fatal(err)
				}
			})

			it("returns true with jvm-application and WEB-INF", func() {
				_, ok, err := base.NewBase(f.Build)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(ok).To(gomega.BeTrue())
			})

			it("links application to ROOT", func() {
				b, _, err := base.NewBase(f.Build)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				g.Expect(b.Contribute()).To(gomega.Succeed())

				layer := f.Build.Layers.Layer("catalina-base")
				g.Expect(filepath.Join(layer.Root, "webapps", "ROOT")).To(test.BeASymlink(f.Build.Application.Root))
			})

			it("links application to BP_TOMCAT_CONTEXT_PATH", func() {
				defer test.ReplaceEnv(t, "BP_TOMCAT_CONTEXT_PATH", "foo/bar")()

				b, _, err := base.NewBase(f.Build)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				g.Expect(b.Contribute()).To(gomega.Succeed())

				layer := f.Build.Layers.Layer("catalina-base")
				g.Expect(filepath.Join(layer.Root, "webapps", "foo#bar")).To(test.BeASymlink(f.Build.Application.Root))
			})

			it("contributes configuration", func() {
				b, _, err := base.NewBase(f.Build)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				g.Expect(b.Contribute()).To(gomega.Succeed())

				layer := f.Build.Layers.Layer("catalina-base")
				g.Expect(filepath.Join(layer.Root, "conf", "context.xml")).To(gomega.BeAnExistingFile())
				g.Expect(filepath.Join(layer.Root, "conf", "logging.properties")).To(gomega.BeAnExistingFile())
				g.Expect(filepath.Join(layer.Root, "conf", "server.xml")).To(gomega.BeAnExistingFile())
				g.Expect(filepath.Join(layer.Root, "conf", "web.xml")).To(gomega.BeAnExistingFile())
			})

			it("contributes access logging support", func() {
				b, _, err := base.NewBase(f.Build)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				g.Expect(b.Contribute()).To(gomega.Succeed())

				layer := f.Build.Layers.Layer("catalina-base")
				g.Expect(filepath.Join(layer.Root, "lib", "stub-tomcat-access-logging-support.jar")).To(gomega.BeAnExistingFile())
				g.Expect(layer).To(test.HaveProfile("access-logging", `ENABLED=${BPL_TOMCAT_ACCESS_LOGGING:=n}

if [[ "${ENABLED}" = "n" ]]; then
	return
fi

printf "Tomcat Access Logging enabled\n"

export JAVA_OPTS="${JAVA_OPTS} -Daccess.logging.enabled=true"
`))
			})

			it("contributes lifecycle support", func() {
				b, _, err := base.NewBase(f.Build)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				g.Expect(b.Contribute()).To(gomega.Succeed())

				layer := f.Build.Layers.Layer("catalina-base")
				g.Expect(filepath.Join(layer.Root, "lib", "stub-tomcat-lifecycle-support.jar")).To(gomega.BeAnExistingFile())
			})

			it("contributes logging support", func() {
				b, _, err := base.NewBase(f.Build)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				g.Expect(b.Contribute()).To(gomega.Succeed())

				layer := f.Build.Layers.Layer("catalina-base")
				destination := filepath.Join(layer.Root, "bin", "stub-tomcat-logging-support.jar")
				g.Expect(destination).To(gomega.BeAnExistingFile())
				g.Expect(filepath.Join(layer.Root, "bin", "setenv.sh")).To(test.HavePermissions(0755))
				g.Expect(filepath.Join(layer.Root, "bin", "setenv.sh")).To(test.HaveContent(fmt.Sprintf(`#!/bin/sh

CLASSPATH=%s`, destination)))
			})

			it("contributes temporary directory", func() {
				b, _, err := base.NewBase(f.Build)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				g.Expect(b.Contribute()).To(gomega.Succeed())

				layer := f.Build.Layers.Layer("catalina-base")
				g.Expect(filepath.Join(layer.Root, "temp")).To(gomega.BeADirectory())
			})

			when("external configuration", func() {

				it("fails with BP_TOMCAT_EXT_CONF_VERSION and no others", func() {
					defer test.ReplaceEnv(t, "BP_TOMCAT_EXT_CONF_VERSION", "test-version")()

					_, _, err := base.NewBase(f.Build)
					g.Expect(err).To(gomega.MatchError("all of $BP_TOMCAT_EXT_CONF_VERSION, $BP_TOMCAT_EXT_CONF_URI, and $BP_TOMCAT_EXT_CONF_SHA256 must be set"))
				})

				it("fails with BP_TOMCAT_EXT_CONF_URI and no others", func() {
					defer test.ReplaceEnv(t, "BP_TOMCAT_EXT_CONF_URI", "test-uri")()

					_, _, err := base.NewBase(f.Build)
					g.Expect(err).To(gomega.MatchError("all of $BP_TOMCAT_EXT_CONF_VERSION, $BP_TOMCAT_EXT_CONF_URI, and $BP_TOMCAT_EXT_CONF_SHA256 must be set"))
				})

				it("fails with BP_TOMCAT_EXT_CONF_SHA256 and no others", func() {
					defer test.ReplaceEnv(t, "BP_TOMCAT_EXT_CONF_SHA256", "test-sha256")()

					_, _, err := base.NewBase(f.Build)
					g.Expect(err).To(gomega.MatchError("all of $BP_TOMCAT_EXT_CONF_VERSION, $BP_TOMCAT_EXT_CONF_URI, and $BP_TOMCAT_EXT_CONF_SHA256 must be set"))
				})

				it("contributes env var external configuration", func() {
					v, err := semver.NewVersion("1.0.0")
					g.Expect(err).NotTo(gomega.HaveOccurred())

					d := buildpack.Dependency{
						ID:      "tomcat-external-configuration",
						Name:    "Tomcat External Configuration",
						Version: buildpack.Version{Version: v},
						URI:     "https://localhost/stub-external-configuration.tar.gz",
						SHA256:  "test-sha256",
						Stacks:  buildpack.Stacks{f.Build.Stack},
						Licenses: buildpack.Licenses{
							{Type: "Proprietary"},
						},
					}

					l := f.Build.Layers.Layer(d.SHA256)
					if err := helper.CopyFile(filepath.Join("testdata", "stub-external-configuration.tar.gz"),
						filepath.Join(l.Root, "stub-external-configuration.tar.gz")); err != nil {
						t.Fatal(err)
					}

					file, err := os.OpenFile(l.Metadata, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
					if err != nil {
						t.Fatal(err)
					}
					defer file.Close()

					if err := toml.NewEncoder(file).Encode(map[string]interface{}{"metadata": d}); err != nil {
						t.Fatal(err)
					}

					defer test.ReplaceEnv(t, "BP_TOMCAT_EXT_CONF_VERSION", d.Version.String())()
					defer test.ReplaceEnv(t, "BP_TOMCAT_EXT_CONF_URI", d.URI)()
					defer test.ReplaceEnv(t, "BP_TOMCAT_EXT_CONF_SHA256", d.SHA256)()

					b, _, err := base.NewBase(f.Build)
					g.Expect(err).NotTo(gomega.HaveOccurred())

					g.Expect(b.Contribute()).To(gomega.Succeed())

					layer := f.Build.Layers.Layer("catalina-base")
					g.Expect(filepath.Join(layer.Root, "fixture-marker")).To(gomega.BeAnExistingFile())
				})

				it("contributes buildpack.toml external configuration", func() {
					f.AddDependency("tomcat-external-configuration", filepath.Join("testdata", "stub-external-configuration.tar.gz"))

					b, _, err := base.NewBase(f.Build)
					g.Expect(err).NotTo(gomega.HaveOccurred())

					g.Expect(b.Contribute()).To(gomega.Succeed())

					layer := f.Build.Layers.Layer("catalina-base")
					g.Expect(filepath.Join(layer.Root, "fixture-marker")).To(gomega.BeAnExistingFile())
				})

			})

			it("sets CATALINA_BASE", func() {
				b, _, err := base.NewBase(f.Build)
				g.Expect(err).NotTo(gomega.HaveOccurred())

				g.Expect(b.Contribute()).To(gomega.Succeed())

				layer := f.Build.Layers.Layer("catalina-base")
				g.Expect(layer).To(test.HaveLayerMetadata(false, false, true))
				g.Expect(layer).To(test.HaveOverrideLaunchEnvironment("CATALINA_BASE", layer.Root))
			})
		})
	}, spec.Report(report.Terminal{}))
}

# `tomcat-cnb`
The Cloud Foundry Tomcat Buildpack is a Cloud Native Buildpack V3 that provides Apache Tomcat to applications that are WAR files.

This buildpack is designed to work in collaboration with other buildpacks.

## Behavior
The buildpack will participate if all of the following conditions are met

* The application is a Java application
* The application has a `WEB-INF/` directory

The buildpack will do the following:

* Contribute a Tomcat home
* Contribute a Tomcat base with the following:
  * `context.xml` from the buildpack root
  * `logging.properties` from the buildpack root
  * `server.xml` from the buildpack root
  * `web.xml` from the buildpack root
  * [Access Logging Support][als]
  * [Lifecycle Support][lcs]
  * [Logging Support][lgs]
  * External Configuration if configured in either `buildpack.toml` or via [environment variables](#Configuration)
  * The application to `webapps/ROOT` unless otherwise [configured](#Configuration)

[als]: https://github.com/cloudfoundry/java-buildpack-support/tree/master/tomcat-access-logging-support
[lcs]: https://github.com/cloudfoundry/java-buildpack-support/tree/master/tomcat-lifecycle-support
[lgs]: https://github.com/cloudfoundry/java-buildpack-support/tree/master/tomcat-logging-support

## Configuration
| Environment Variable | Description
| -------------------- | -----------
| `$BP_TOMCAT_CONTEXT_PATH` | The context path to mount the application at.  Defaults to empty (`ROOT`).
| `$BP_TOMCAT_EXT_CONF_SHA256` | The SHA256 hash of the external configuration package
| `$BP_TOMCAT_EXT_CONF_URI` | The download URI of the external configuration package
| `$BP_TOMCAT_EXT_CONF_VERSION` | The version of the external configuration package
| `$BP_TOMCAT_VERSION` | Semver value of the version of Tomcat to use.  Defaults to `9.*`.

### External Configuration Package
he artifacts that the repository provides must be in TAR format and must follow the Tomcat archive structure:

```
tomcat
└── conf
    ├── context.xml
    ├── server.xml
    ├── web.xml
    ├── ...
```

## Detail
* **Requires**
  * `jvm-application`

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: https://www.apache.org/licenses/LICENSE-2.0

/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package configs

/**
This is an initial implementation of Config. Please keep in mind we're still working to improve how we handle
environment variables and general config data.
*/

const DefaultK1Version = "development"

// K1Version is used on version command. The value is dynamically updated on build time via ldflag. Built Kubefirst
// versions will follow semver value like 1.9.0, when not using the built version, "development" is used.
var K1Version = DefaultK1Version

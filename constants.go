package pipenvinstall

// SitePackages is the name of the dependency provided by the Pipenv Install
// buildpack.
const SitePackages = "site-packages"

// CPython is the name of the python runtime dependency provided by the CPython
// buildpack: https://github.com/paketo-buildpacks/cpython.
const CPython = "cpython"

// Pipenv is the name of the dependency provided by the Pipenv buildpack:
// https://github.com/paketo-buildpacks/pipenv.
const Pipenv = "pipenv"

// The layer name for packages layer. This layer is where dependencies are
// installed to.
const PackagesLayerName = "packages"

// The layer name for cache layer. This layer holds the pipenv cache.
const CacheLayerName = "cache"

# Omniverse

Omniverse allows to substitute strings in a set of files

## Install

### Binary Download

Navigate to [Releases](https://github.com/unprofession-al/omniverse/releases), grab
the package that matches your operating system and architecture. Unpack the archive
and put the binary file somewhere in your `$PATH`

### From Source

Make sure you have [go](https://golang.org/doc/install) installed, then run: 

```
# go get -u https://github.com/unprofession-al/omniverse
```

## Configure

Create a configuration file that looks like this:

```yaml
singularity:
  expression: \<\<\W*([a-zA-Z0-9_]+)\W*\>\>
  expression_template: << {{.}} >>
alterverses:
  prod:
    env: Production
    loadbalancer: prod.lb.example.com
  dev:
    env: Development
    loadbalancer: dev.lb.example.com
```

## Run

```bash
# omniverse                                         
Create a copy of a directory with deviations

Usage:
  omniverse [command]

Available Commands:
  create-alterverse     Create alterverse from singularity
  deduce-singularity    Deduce singularity from alterverse
  help                  Help about any command
  list-singularity-keys Discover and list keys which are defined in singularity
  print-config          Print the configuration as parsed by omniverse

Flags:
  -c, --config string        configuration file for omniverse (default "omniverse.yaml")
  -h, --help                 help for omniverse
  -q, --quiet                omit log output
  -s, --singularity string   path of the singularity (default "singularity")

Use "omniverse [command] --help" for more information about a command.
```

## Next up

* Better validation of the configuration: 
  * Does the `expression` with the `expression_template` rendered with all definitions?
* `watch` option
* Implement `ignore` list
* When deducing the singularity, check if the source already has strings that match the expression

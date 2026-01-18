# Omniverse

Omniverse allows to substitute strings in a set of files

## Key Feature

### String substitution

Omniverse is a tiny command line tool. Its main purpose is to perform string
substitution on multiple files in a correct manner. It does so by reading a source
directory and its configuration file (`manifest`) and creating an altered copy of
the directories content in a destination directory using another `manifest`. 

The `manifests` are simple mappings of the strings that are required to substitute.

### That's it

Omniverse does nothing other than that: No complex templating or other logic is
applied other than string substitution. This is considered a feature.

### When to Use (and When Not To)

Omniverse is intentionally simple and works best for specific use cases. However, in many situations
other approaches are preferable:

- **Terraform modules**: If you're managing infrastructure as code, proper modularization with input
  variables is usually the better choice. It provides type safety, validation, and better tooling support.
- **Helm charts / Kustomize**: For Kubernetes configurations, these tools offer environment-specific
  overlays with proper schema validation.
- **Configuration management tools**: Ansible, Puppet, or Chef provide templating with logic, facts,
  and proper secret management.
- **Environment variables**: For application configuration, 12-factor style environment variables
  are often simpler and more portable.
- **Template engines**: Tools like `envsubst`, Jinja2, or Go templates offer more control when you
  need conditional logic or loops.

Omniverse shines when you need to maintain parallel copies of existing files with minimal changes to
the original workflow -- for example, when retrofitting environment separation onto a codebase that
wasn't designed for it, or when the overhead of proper tooling isn't justified.

## Install

### Binary Download

Navigate to [Releases](https://github.com/unprofession-al/omniverse/releases), grab
the package that matches your operating system and architecture. Unpack the archive
and put the binary file somewhere in your `$PATH`

### From Source

Make sure you have [go](https://golang.org/doc/install) installed, then run:

```bash
# go get -u https://github.com/unprofession-al/omniverse
```

## Configuration

Omniverse takes an input directory and an output directory as arguments. Both of
these directories need to have a `manifest` file in its root. These files must
be named `.alterverse.yml`. The manifests are written in YAML markup and must look
similar to this:

```yaml
---
manifest:
  env: production
  loadbalancer: prod.lb.example.com
```

Given the example from above is located in the root uppermost directory of the
source directory and the destination directory contains a `.alterverse.yml` file
with the following content:

```yaml
---
manifest:
  env: test
  loadbalancer: test.lb.example.com
```

An arbitrary file present in the source folder with the content of...

```terraform
resource "aws_lb" "production" {
  name               = "production_lb"
  internal           = false
  load_balancer_type = "network"
  subnets            = ["${aws_subnet.public.*.id}"]

  tags = {
    Environment = "production"
    DNSRecord = "prod.lb.example.com"
  }
}
```

... would be rendered to the destination directory with the content of...

```terraform
resource "aws_lb" "test" {
  name               = "test_lb"
  internal           = false
  load_balancer_type = "network"
  subnets            = ["${aws_subnet.public.*.id}"]

  tags = {
    Environment = "test"
    DNSRecord = "test.lb.example.com"
  }
}
```

## Run

```bash
# omniverse help
Create a copy of a directory with deviations

Usage:
  omniverse [command]

Available Commands:
  deduce      Deduce an alterverse
  help        Help about any command
  version     Print version info

Flags:
  -h, --help   help for omniverse

Use "omniverse [command] --help" for more information about a command.
```

To execute the example from the _Configuration_ section run:

```
omniverse deduce --from /tmp/prod --to /tmp/test
```

## Further Reading

For a deeper dive into the theory behind text substitution and the edge cases that `omniverse` handles,
see [IDEA.md](IDEA.md).

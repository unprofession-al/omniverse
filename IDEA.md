---
{
    "title": "On Text Substitution",
    "subtitle": "The \"Complex\" Details of a Simple Process",
    "date": "2019-01-22",
    "author": "Daniel Menet",
}
---

Text replacement is probably one of the most widely used things a computer can do for us: Bulk letters, the early days 
of configuration management, code refactoring, basic template rendering... string substitution is used everywhere. As an
IT infrastructure person beloved tools like `sed` and `awk` make a lot of my work easier. `:%s/.../.../g` is one of the few 
`vim` commands I will hardly ever forget. And I maintain a strong hate/love relationship with
[regular expressions](https://en.wikipedia.org/wiki/Regular_expression).

## A Case for Text Substitution

_This section explains why this topic came up in the first place. Feel free to skip..._

Even with all the tooling we have today text substitution has a relevant place in our daily business. Here's an 
example: Some colleagues set up an account on AWS using [terraform](https://www.terraform.io/) and deploying a few 
hundred resources. Shortly after having the account in place we decided that a replica of the account with all its
resources is required for infrastructure testing, integration of new software releases etc. The replication of the
account implicated that the variable parts of each account must be managed in some way. This included:

- some "variables" sprinkled all over the `terraform` repository (containing about 170 files in 25 directories)
- changes to a bunch of configuration files which are basically pushed from a git repository to S3 and from there
  injected into the EC2 instances at boot time.
- a few changes to a repository containing the [packer](https://www.packer.io/) configurations

For each part there is an ideal solution. For the `terraform` stuff for example a way would be to create modules
which then could be reused for both the accounts. In case of the configuration files we could have a configuration
management system such as Puppet or Ansible set up. 

However we decided not to do so due to various reasons:

- For every component listed above a separate solution would be required.
- Each solution would require some engineering effort to achieve a technically small requirement.
- The work flow to apply changes to one environment, test and then copy the changes to the production
  would gain a decent amount of complexity.
- A fair amount of additional tooling would be required.
- The current setup is quite simple; you can read through the `terraform` repository and understand exactly what
  resources are deployed. By introducing an extra level of abstraction (as in `terraform` modules) this quality
  will be lost to some extent.

The options we were left with were _templating_ and _search-and-replace_. Although the _templating_ stuff
feels tempting ("It allows us to do more stuff that `terraform` is currently not capable to do!" etc.) I am
not a devotee of this solution since it would introduce complexity on a layer that we currently don't even
have. Also this would perturb the work flow of for example change a line and the run `terraform plan` to see
its outcome. We would rather have to fiddle with the template, run the rendering engine and then `terraform
plan`. So no.

Regarding all of this, what could possibly go wrong with a good old bash script that iterates over the files
concerned, calls `sed` for each string to be replaced and writes the files to a new directory...

## The Theory

As you probably know a couple of things can go wrong. Here are some of the problems and possible solutions.
Let's look at the most obvious solution. 

First we need some sort of configuration file which tells the script how the replacement should occur because 
we certainly do not want to hard-code this. Lets pick a format which we can easily process. Since our script
will be written in `bash` for demo purposes we use a format that defines a set of data per line and can be
`eval`ed in order to be used as variables. Here's our `replacer.manifest` file:

```
FROM=example.com TO=example-int.com
FROM=production  TO=integration
```

Here's the first iteration of our script (let's call it `replacer.sh`) -- pretty straight forward:

``` bash
#!/bin/bash

CONFIG="$1"
FILE="$2"

>&2 echo "Processing $FILE with data read from $CONFIG..."
>&2 echo -----

# creating a temp file
TMPFILE=$(mktemp /tmp/$FILE.XXXXXX)
cp $FILE $TMPFILE

# reading the config file
LOOKUP=()
while IFS= read -r LINE; do
  LOOKUP+=("$LINE")
done < "$CONFIG"

# replacing the strings
for TOREPLACE in "${LOOKUP[@]}"; do
    eval $TOREPLACE
    >&2 echo "Replacing '${FROM}' with '${TO}'"
    sed -i -e "s/${FROM}/${TO}/g" $TMPFILE
done
>&2 echo -----

# printing to STDOUT and removing temp file
cat "$TMPFILE"
rm $TMPFILE
```

The script takes two arguments: the first is our configuration, the second is the file to process.
A few messages are printed to STDERR, the processed file is printed to STDOUT.

We also need a test file which we can use to test. For this iteration create a file named `README.md` 
with the following content:

```
This is the __production__ environment.
The API can be reached at `api.example.com`.
```

Let's run our script:

```
# ./replacer.sh replacer.manifest README.md
Processing README.md with data read from replacer.manifest...
-----
Replacing 'example.com' with 'example-int.com'
Replacing 'production' with 'integration'
-----
This is the __integration__ environment.
The API can be reached at `api.example-int.com`.
```

__Success!__ For now...

### Switching strings

Next let's switch to an edge case: Given the `README.md` has changed a bit:

```
This is the repository of the production environment (example.com).
All API calls to its integration environment (example-int.com) must be avoided.
```

This certainly also needs some adaption in our `replacer.manifest` since we not only want to ensure that 
for example `production` will be changed to `integration` but also vice versa:

```
FROM=example.com     TO=example-int.com
FROM=example-int.com TO=example.com
FROM=production      TO=integration
FROM=integration     TO=production
```

We already have our script, so off we go:

```
Processing README.md with data read from replacer.manifest...
-----
Replacing 'example.com' with 'example-int.com'
Replacing 'example-int.com' with 'example.com'
Replacing 'production' with 'integration'
Replacing 'integration' with 'production'
-----
This is the repository of the production environment (example.com).                                        
All API calls to its production environment (example.com) must be avoided.
```

Well, this obviously did not work as expected; Apparently _switching_ strings from one environment to another
is not as straight forward. How should our script know which strings are already replaced?

__Solution:__ This can be simply solved with an intermediate step: We first replace the string with a place holder
(which must be unique) for each string and can only then -- in a second iteration -- replace those place holders 
with the target string (mighty tools like `awk` can probably do this in one line but you get the idea).

``` bash
#!/bin/bash

CONFIG="$1"
FILE="$2"

>&2 echo "Processing $FILE with data read from $CONFIG..."
>&2 echo -----

# creating a temp file
TMPFILE=$(mktemp /tmp/$FILE.XXXXXX)
cp $FILE $TMPFILE

# reading the config file
LOOKUP=()
while IFS= read -r LINE; do
    LOOKUP+=("$LINE UUID=$(uuidgen)")
done < "$CONFIG"

# Substitute 'from' with 'uuid' first
for TOREPLACE in "${LOOKUP[@]}"; do
    eval $TOREPLACE
    >&2 echo "Replacing '${FROM}' with '${UUID}'"
    sed -i -e "s/${FROM}/${UUID}/g" $TMPFILE
done
>&2 echo -----

# Substitute 'uuid' with 'to' first
for TOREPLACE in "${LOOKUP[@]}"; do
    eval $TOREPLACE
    >&2 echo "Replacing '${UUID}' with '${TO}'"
    sed -i -e "s/${UUID}/${TO}/g" $TMPFILE
done
>&2 echo -----

# printing to STDOUT and removing temp file
cat "$TMPFILE"
rm $TMPFILE
```

Run it:

```
Processing README.md with data read from replacer.manifest...
-----
Replacing 'example.com' with '4484fbbd-b0ec-49b3-a73c-dc7fda23133b'
Replacing 'example-int.com' with '1adc5f75-83d3-416d-ba00-8ce752a88ade'
Replacing 'production' with '88dad809-da4c-45b3-afa8-5903386f3593'
Replacing 'integration' with 'f15499c3-cfc3-4562-ae7b-fc1236f55872'
-----
Replacing '4484fbbd-b0ec-49b3-a73c-dc7fda23133b' with 'example-int.com'
Replacing '1adc5f75-83d3-416d-ba00-8ce752a88ade' with 'example.com'
Replacing '88dad809-da4c-45b3-afa8-5903386f3593' with 'integration'
Replacing 'f15499c3-cfc3-4562-ae7b-fc1236f55872' with 'production'
-----
This is the repository of the integration environment (example-int.com).                                        
All API calls to its production environment (example.com) must be avoided.
```

Now we have a correct result. But there is more...

### Replacing Strings that are Substrings of others first

Now let's look at the next pitfall: If you need to replace strings that are substrings of other strings
things can get tricky again. Given the following situation:

```
FROM=example.com     TO=example-int.com
FROM=api.example.com TO=next-api.example-int.com
FROM=production      TO=integration
```

Our `README.md` has changed again:

```
The production environment consists of a series of HTTP endpoints exposed to the internet:

- An end user website is presented at www.example.com and example.com respectively.
- A management frontend is accessible via admin.example.com.
- An API is exposing functionality at api.example.com.
```

Running our script exposes the next issue:

```
# ./replacer.sh replacer.manifest README.md
Processing README.md with data read from replacer.manifest...
-----
Replacing 'example.com' with 'e84dc1b2-9e9d-4626-8813-07e93ec2b560'
Replacing 'api.example.com' with 'fd1c475f-7bca-480a-a9a3-95cd46a30c0a'
Replacing 'production' with '016dd3db-eea7-4e1c-990e-d521e20e47ae'
-----
Replacing 'e84dc1b2-9e9d-4626-8813-07e93ec2b560' with 'example-int.com'
Replacing 'fd1c475f-7bca-480a-a9a3-95cd46a30c0a' with 'next-api.example-int.com'
Replacing '016dd3db-eea7-4e1c-990e-d521e20e47ae' with 'integration'
-----
The integration environment consists of a series of HTTP endpoints exposed to the internet:

- An end user website is presented at www.example-int.com and example-int.com respectively.
- A management frontend is accessible via admin.example-int.com.
- An API is exposing functionality at api.example-int.com.
```

On the last line we expected the domain name to be `next-api.example-int.com` but we got `api.example-int.com`. What
happened is that during the process the substitution of `example.com` came first. As the turn was on `api.example.com`
the last line already was substituted and the domain string already was `api-e84dc1b2-9e9d-4626-8813-07e93ec2b560` in
that particular case -- `api.example.com` does not match that.

__Solution:__ Again the solution is rather simple: Before we start replacing strings we need to order our lookup table
by the `FROM` field: The longer the string the earlier the replacement must occur. We can now throw `awk` into the mix --
Let's adapt the script:

``` bash
#!/bin/bash

CONFIG="$1"
FILE="$2"

>&2 echo "Processing $FILE with data read from $CONFIG..."
>&2 echo -----

# creating a temp file
TMPFILE=$(mktemp /tmp/$FILE.XXXXXX)
cp $FILE $TMPFILE

# sort the FROM keys in the manifest, the long ones first
SORTEDMANIFEST=$(mktemp /tmp/$CONFIG.XXXXXX)
cat $CONFIG | awk '{ print length($1), $0 }' | sort -nr | cut -d" " -f2- > $SORTEDMANIFEST

# reading the config file
LOOKUP=()
while IFS= read -r LINE; do
    LOOKUP+=("$LINE UUID=$(uuidgen)")
done < "$SORTEDMANIFEST"

# Substitute 'from' with 'uuid' first
for TOREPLACE in "${LOOKUP[@]}"; do
    eval $TOREPLACE
    >&2 echo "Replacing '${FROM}' with '${UUID}'"
    sed -i -e "s/${FROM}/${UUID}/g" $TMPFILE
done
>&2 echo -----

# Substitute 'uuid' with 'to' first
for TOREPLACE in "${LOOKUP[@]}"; do
    eval $TOREPLACE
    >&2 echo "Replacing '${UUID}' with '${TO}'"
    sed -i -e "s/${UUID}/${TO}/g" $TMPFILE
done
>&2 echo -----

# printing to STDOUT and removing temp file
cat "$TMPFILE"

# cleanup
rm $TMPFILE $SORTEDMANIFEST
```

The interesting part of this iteration happens on the line `cat $CONFIG | awk '{ print length($1), $0 }' | 
sort -nr | cut -d" " -f2- > $SORTEDMANIFEST`. If you are not exactly sure what all that means just run the 
commands step by step to see the output after each `pipe`.

Let's check again:

```
# /replacer.sh replacer.manifest README.md
Processing README.md with data read from replacer.manifest...
-----
Replacing 'api.example.com' with '49c87003-8faa-4810-b197-24aadf654452'
Replacing 'example.com' with '06ee395f-e1c2-403a-a5b4-f8395f03a796'
Replacing 'production' with '9fd63e00-a8cd-4860-82e9-e0c0324115f7'
-----
Replacing '49c87003-8faa-4810-b197-24aadf654452' with 'next-api.example-int.com'
Replacing '06ee395f-e1c2-403a-a5b4-f8395f03a796' with 'example-int.com'
Replacing '9fd63e00-a8cd-4860-82e9-e0c0324115f7' with 'integration'
-----
The integration environment consists of a series of HTTP endpoints exposed to the internet:

- An end user website is presented at www.example-int.com and example-int.com respectively.
- A management frontend is accessible via admin.example-int.com.
- An API is exposing functionality at next-api.example-int.com.
```

Sweet, that's it!

### The final challenge

The next iteration of our script is not really an improvement to its logic but rather some handling of
a likely-to-happen user error. Considering the fact that we now have a script that does its job quite 
well and allows us to move fast our `replacer.manifest` will most likely grow. At some point we will
have a doublet of a FROM field in our configuration:

```
FROM=example.com     TO=www.example-int.com
FROM=production      TO=integration
FROM=example.com     TO=example-int.com
FROM=api.example.com TO=next-api.example-int.com
```

Here somebody stopped caring a bit too hard and introduced a new replacement set on the first line for the
web server. Since the web server can be accessed via `www.example.com` and `example.com` the person got
confused and added some extra mapping -- which now overlaps with the third line.

Let's see how this results by applying the changes to the `README.md` from the last example:

```
# ./replacer.sh replacer.manifest README.md
Processing README.md with data read from replacer.manifest...
-----
Replacing 'api.example.com' with '3d68d628-b749-4587-89b4-415f1a2b28a2'
Replacing 'example.com' with '8b8bc6f9-0f9e-4729-8201-7be4bab3637b'
Replacing 'example.com' with '25e943d2-a2d3-4601-a2b7-0e53e44ec701'
Replacing 'production' with 'd4ba475d-8bee-4c87-8360-6dab990de13a'
-----
Replacing '3d68d628-b749-4587-89b4-415f1a2b28a2' with 'next-api.example-int.com'
Replacing '8b8bc6f9-0f9e-4729-8201-7be4bab3637b' with 'www.example-int.com'
Replacing '25e943d2-a2d3-4601-a2b7-0e53e44ec701' with 'example-int.com'
Replacing 'd4ba475d-8bee-4c87-8360-6dab990de13a' with 'integration'
-----
The integration environment consists of a series of HTTP endpoints exposed to the internet:

- An end user website is presented at www.www.example-int.com and www.example-int.com respectively.
- A management frontend is accessible via admin.www.example-int.com.
- An API is exposing functionality at next-api.example-int.com.
```

As expected this got a little funky. By accident the pair that substitutes `example.com` with `www.example-int.com`
did run before the one that would have placed just `example-int.com`. Now the `www` is everywhere!

__Solution:__ Well, there is no solution to this: If two _search strings_ are equal our script has no chance
to guess in which order to substitute these. Therefore the best we can do is to throw an error and exit before
bad things happen. Let's fix the script:

``` bash
#!/bin/bash

CONFIG="$1"
FILE="$2"

# check for doublets in the manifest
DOUBLETS=$(cat $CONFIG | awk '{print $1}' | sort | uniq -d)
if [ ! -z "$DOUBLETS" ]; then
    >&2 echo "ERROR: Some keys are present more than once in the config file $CONFIG:"
    >&2 echo "$DOUBLETS"
    exit 255
fi

>&2 echo "Processing $FILE with data read from $CONFIG..."
>&2 echo -----

# creating a temp file
TMPFILE=$(mktemp /tmp/$FILE.XXXXXX)
cp $FILE $TMPFILE

# sort the FROM keys in the manifest, the long ones first
SORTEDMANIFEST=$(mktemp /tmp/$CONFIG.XXXXXX)
cat $CONFIG | awk '{ print length($1), $0 }'  | sort -nr | cut -d" " -f2- > $SORTEDMANIFEST

# reading the config file
LOOKUP=()
while IFS= read -r LINE; do
    LOOKUP+=("$LINE UUID=$(uuidgen)")
done < "$SORTEDMANIFEST"

# Substitute 'from' with 'uuid' first
for TOREPLACE in "${LOOKUP[@]}"; do
    eval $TOREPLACE
    >&2 echo "Replacing '${FROM}' with '${UUID}'"
    sed -i -e "s/${FROM}/${UUID}/g" $TMPFILE
done
>&2 echo -----

# Substitute 'uuid' with 'to' first
for TOREPLACE in "${LOOKUP[@]}"; do
    eval $TOREPLACE
    >&2 echo "Replacing '${UUID}' with '${TO}'"
    sed -i -e "s/${UUID}/${TO}/g" $TMPFILE
done
>&2 echo -----

# printing to STDOUT and removing temp file
cat "$TMPFILE"

# cleanup
rm $TMPFILE $SORTEDMANIFEST
```

Run it again:

```
./replacer.sh replacer.manifest README.md
ERROR: Some keys are present more than once in the config file replacer.manifest:
FROM=example.com
```

Disaster averted! Note that we would also have to do the same check for the `TO` keys in order to be able
to convert our files back and forth.

### The other side of the coin: Duplicate Values

As hinted earlier we should also check for duplicates on the `TO` side. While the reasoning for checking `FROM`
keys is obvious (the script wouldn't know which replacement to use), the `TO` side is a bit more subtle.

Consider this manifest:

```
FROM=api.example.com TO=api.example-int.com
FROM=www.example.com TO=api.example-int.com
FROM=production      TO=integration
```

Notice that both `api.example.com` and `www.example.com` map to the same target `api.example-int.com`. Running
our script would work just fine -- all occurrences would be replaced as specified. The problem only surfaces
when we try to go back: How would our script know whether `api.example-int.com` should become `api.example.com`
or `www.example.com`? It can't.

If we want our substitution to be _bidirectional_ (and we do, otherwise we'd lose the ability to sync changes
back), we must ensure that values are unique on both sides of our manifest. This is essentially the same check
we already have, just applied to the `TO` column as well.

### Pre-existing Destination Values

Here's another subtle trap. Let's say our manifest looks perfectly fine:

```
FROM=foo TO=bar
FROM=baz TO=qux
```

And our source file contains:

```
The foo service talks to the bar service.
```

Running the substitution gives us:

```
The bar service talks to the bar service.
```

The replacement worked correctly. But now try to reverse it: we'd replace `bar` with `foo`, resulting in:

```
The foo service talks to the foo service.
```

That's not what we started with! The problem is that our _target_ value `bar` already existed in the source
file before we even started replacing. When reversing, the script cannot distinguish between the `bar` that
was originally `foo` and the `bar` that was always `bar`.

__Solution:__ Before performing any substitution, we should verify that none of the `TO` values already exist
in the source files. If they do, the substitution might work, but reversibility is compromised. In such cases
it's better to warn the user and let them decide how to proceed.

### Verifying Reversibility

So far we've added checks to _prevent_ irreversible situations. But what if we want to be absolutely certain
that a substitution can be reversed? The most reliable way is to simply try it.

The idea is straightforward:

1. Convert from A to B (source → destination)
2. Convert from B back to A (destination → source)
3. Compare the result with the original

If the round-trip produces identical content, we have proof that the conversion is reversible. If not, something
went wrong -- perhaps a `TO` value was already present in the source, or there's some edge case we didn't
anticipate.

This verification adds overhead since we're essentially doing the work twice, but for critical operations it
provides a guarantee that no information is lost. In practice, you might want to offer both a "fast" mode
(just do the substitution) and a "strict" mode (verify reversibility).

## In Practical Use

So far we have looked at a few logical problems that can easily be prevented -- as shows with the iterations
of our script. But how about mistakes that we cannot prevent during execution? Our users (including ourself)
tend to do stupid stuff sometimes. Can we determine some _metrics_ and define 
[_code smells_](https://en.wikipedia.org/wiki/Code_smell) that help us discover those mistakes?

### Replacing Strings that appear in many Contexts

An obvious mistake would be to search and replace tiny strings that are present as substrings in many other
strings. Let's assume we want to change the location of some sort of deployment from California (AWS region
`us-west-1`) to Dublin (AWS region `eu-west-1`). Consider the following `terraform` file:

```
provider "aws" {
  region  = "us-west-1"
}

# ...

resource "aws_iam_user" "brutus" {
  name = "brutus"
}
```

Since we are all lazy we keep the `replacer.manifest` short:

```
FROM=us TO=eu
```

Easy! Run the script and done...

```
# ./replacer.sh replacer.manifest production.tf 
Processing production.tf with data read from replacer.manifest...
-----
Replacing 'us' with '8e0e3f6c-ec88-4aa2-9406-fa865b5831c6'
-----
Replacing '8e0e3f6c-ec88-4aa2-9406-fa865b5831c6' with 'eu'
-----
provider "aws" {
  region  = "eu-west-1"
}

# ...

resource "aws_iam_euer" "bruteu" {
  name = "bruteu"
}
```

Well the script did work fine, but we messed up: The string `us` seems to appear in a few other _context_ than 
we expected. Unfortunately there's no way to prevent this. But still it would be helpful to 'measure' how wide the
impact of our change really is. To do this we could build a regular expression looking for strings (or _words_) 
that contain our fragment. Here's a new script `findcontexts.sh` to do this:

```
#!/bin/bash
CONFIG="$1"
FILE="$2"

# reading the config file
LOOKUP=()
while IFS= read -r LINE; do
    LOOKUP+=("$LINE")
done < "$CONFIG"

for TOREPLACE in "${LOOKUP[@]}"; do
    eval $TOREPLACE
    echo "Searching contexts of '${FROM}'..."
    cat $FILE | grep -oP '(\b[\w-_]*'$FROM'[\w-_]*\b)' | sort -u
done
```

Let's run it:

```
# ./findcontexts.sh replacer.manifest production.tf 
Searching contexts of 'us'...
aws_iam_user
brutus
us-west-1
```

That would have helped.

Note that in this example the string fragment in the middle of the regular expression (eg. the `$FROM`) part
works only in very simple cases such as the one we looked at. For more complex strings the fragment would be
required to be properly escaped to conform the regular expression.

## Introducing `omniverse`

All the theory above is nice, but maintaining a bash script that grows with every edge case we discover is
not exactly fun. Also, iterating over files with `sed` doesn't scale well when you have hundreds of files
across dozens of directories. Time to build a proper tool.

[`omniverse`](https://github.com/unprofession-al/omniverse) is a small command-line utility written in Go
that implements everything we've discussed -- and a bit more. The core concept revolves around what I call
an _alterverse_: a directory containing files and a manifest (`.alterverse.yml`) that defines the variable
parts for that particular "universe".

Here's what a manifest looks like:

```yaml
manifest:
  domain: example.com
  api_domain: api.example.com
  environment: production
```

The idea is simple: you have a _source_ alterverse (say, your production configuration) and a _destination_
alterverse (your integration environment). Both share the same manifest _keys_ but have different _values_.
Running `omniverse deduce` reads the source, performs all the string substitutions based on the two manifests,
and writes the result to the destination.

Under the hood, `omniverse` implements all the safeguards we've discussed:

- **Longest-first replacement**: The lookup table is sorted by value length to prevent substring conflicts.
- **Tokenizer-based substitution**: Instead of running multiple `sed` passes, `omniverse` tokenizes the content
  and performs all replacements in a single pass. This is both faster and allows for verification.
- **Duplicate value detection**: Both source and destination manifests are validated to ensure no duplicate
  values exist within each manifest.
- **Strict mode**: The `--strict` flag enables round-trip verification. It checks that destination values don't
  already exist in source files and verifies that converting A→B→A produces identical content.
- **Context detection**: A hidden `contexts` command helps identify all the "words" containing your manifest
  values -- useful for catching overly broad replacements before they cause trouble.

### File Synchronization

Beyond string replacement, `omniverse` also handles the mundane but important task of keeping directories
in sync. When files are removed from the source, they should also disappear from the destination. When files
become shorter after substitution, the destination file needs to be properly truncated (a surprisingly easy
thing to get wrong). Hidden files and the manifest itself are excluded by default via configurable ignore
patterns.

### A Typical Workflow

In practice, the workflow looks something like this:

1. Set up your source directory with its manifest defining the "production" values
2. Create a destination directory with a manifest defining the "integration" values
3. Run `omniverse deduce --source ./production --destination ./integration --strict`
4. Review the changes with `git diff` (you _are_ using version control, right?)
5. Make changes in whichever environment makes sense
6. Sync back if needed by swapping source and destination

The bidirectional nature is key: you're not locked into always making changes in one place. Fix a bug in
integration, verify it works, then sync back to production. Or the other way around. As long as your manifests
are set up correctly, `omniverse` ensures nothing gets lost in translation.

## A Word of Caution

Before reaching for string substitution, it's worth asking: is this the right tool for the job? In many cases,
the answer is no. Here are some alternatives that are often preferable:

**Terraform modules**: If you're working with infrastructure as code, the proper solution is usually to
refactor into modules with input variables. Yes, it requires more upfront work, but you get type checking,
validation, documentation, and the full power of the Terraform ecosystem. The "you can read through the
repository and understand exactly what resources are deployed" argument from earlier cuts both ways -- with
modules, you trade some immediate readability for better maintainability and reusability.

**Helm charts and Kustomize**: For Kubernetes manifests, these tools were designed specifically for the
problem of environment-specific configuration. They provide proper schema validation, inheritance, and
a well-understood deployment model.

**Configuration management tools**: Ansible, Puppet, Chef, and SaltStack all offer templating with conditional
logic, fact-based customization, proper secret management, and idempotent execution. If your use case involves
configuring running systems, these are almost certainly better choices.

**Template engines**: Tools like `envsubst`, Jinja2, Mustache, or Go's `text/template` give you explicit
control over what gets substituted. The syntax makes it obvious where variables are used, and you can add
conditional logic when needed.

**Environment variables**: For application configuration, the twelve-factor app approach of using environment
variables is often simpler, more portable, and better supported by deployment platforms.

So when _does_ string substitution make sense? A few scenarios:

- **Retrofitting**: You have an existing codebase that wasn't designed for multi-environment deployment, and
  the cost of proper refactoring exceeds the benefit.
- **Minimal tooling**: You want to avoid introducing new dependencies or changing established workflows.
- **Temporary solutions**: You need something quick while planning a proper migration to better tooling.
- **Simplicity**: The configuration differences are truly just a handful of string values, and adding a
  template layer would be overkill.

The key is to be honest about whether you're choosing string substitution because it's the right tool, or
because it's the easy tool. Sometimes easy is fine. Sometimes it's technical debt in disguise.

## Conclusion

Text substitution seems trivial until you actually try to do it correctly. What starts as a simple `sed`
command quickly spirals into a maze of edge cases: switching strings that replace each other, substrings
that get mangled, duplicate mappings that break reversibility, and target values that already exist in
your source files.

The good news is that all these problems are solvable with a bit of careful engineering:

- Use intermediate placeholders (UUIDs) to handle bidirectional swaps
- Sort replacements by length, longest first
- Validate manifests for duplicate values on both sides
- Check for pre-existing destination values in source files
- Verify reversibility by actually performing a round-trip

`omniverse` packages all of this into a single tool that's fast enough for everyday use and strict enough
for when you really can't afford to mess things up. It won't solve every configuration management problem
out there, but for the specific use case of maintaining parallel environments with minimal ceremony, it
does the job quite well.

Sometimes the simplest solution really is the best one -- you just have to make sure you understand all
the ways it can go wrong.

## Some Closing Remarks

A few practical tips from running this approach in production:

* **Always use version control**: This should go without saying, but `git diff` is your best friend when
  reviewing substitution results. Commit your destination directory and inspect changes carefully before
  pushing anything to production.

* **Keep manifests minimal**: It's tempting to throw every variable into the manifest, but resist the urge.
  The more entries you have, the higher the chance of unintended matches. Only include values that actually
  differ between environments.

* **Be specific with values**: `us` is a terrible manifest value; `us-west-1` is much better. The more
  unique your values, the fewer false positives you'll encounter. When in doubt, use the `contexts` command
  to check what you're about to replace.

* **Test the round-trip**: Before relying on bidirectional sync, actually test it. Make a change in A, sync
  to B, sync back to A. If the result is identical to your original change, you're good. If not, your manifest
  needs work.

* **Watch out for generated files**: Build artifacts, compiled assets, and other generated files can contain
  surprising string matches. Consider excluding them via ignore patterns or keeping them out of your
  alterverse directories entirely.

* **Document your manifests**: A comment explaining _why_ each value needs to differ between environments
  saves future-you (and your colleagues) from accidentally removing something important.

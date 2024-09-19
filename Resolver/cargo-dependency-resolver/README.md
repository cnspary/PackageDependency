
# Cargo Dependency Resolver

### Build projects
Change `registry` path in `./.cargo/config.toml` to absolute path to your `crates.io-index`.

```
[net]
git-fetch-with-cli = true

[source.mirror]
registry = "file:///absolute/path/to/crates.io-index/dir"

[source.crates-io]
replace-with = "mirror"
```

### Running projects
#### 1. Run a single deps

```Shell
cargo run --bin get_deps <name> <version_num>
```
The output format follows: `<dep_name>,<version_num>,<dep_depth>`. 

Output example:
```bash
cargo run --bin get_deps rand 0.8.5

unicode-ident,1.0.3,4
unicode-ident,1.0.3,7
syn,1.0.99,3
cfg-if,1.0.0,2
proc-macro2,1.0.43,3
rand_chacha,0.3.1,1
......

```

#### 2. Dependency graph
If you want to view full dependency info, you can run
```Shell
cargo run --bin get_deps full <name> <version_num>
```
The output format follows: `graph:{ dependenies of each package version} features:{ package features enabled by each package version}`. 
Output example:
```bash
cargo run --bin full get_deps rand 0.8.5`

graph: Graph {
  - cfg-if v1.0.0
  - dep v0.1.0  ## Our virtual package (dep). Its dependencies are direct dependencies while others are indirect dependencies.
    - rand v0.8.5
  - getrandom v0.2.7
    - cfg-if v1.0.0
    - libc v0.2.129
    - wasi v0.11.0+wasi-snapshot-preview1
  - libc v0.2.129
  ......
}
```


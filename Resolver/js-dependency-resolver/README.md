# NPM Dependency Resolver

## Usage

Build the resolver, then run it.
```bash
cmake -B build/release -D CMAKE_BUILD_TYPE=Release
cmake --build build/release
```

The program will:

1. Connect to CounchDB.
2. Fetch the package dependencies.
3. Build a dependency graph and print the dependency graph.

## Note
You should see and edit the config.hpp before building.

***The default main.cpp will not get input and output.*** 
***YOU MUST RUN THE FUNCTION ON YOUR OWN DEMAND.***

## Example Output

```json
{
    "mocha@10.0.0": {
        "@ungap/promise-all-settled@1.1.2": {},
        "ansi-colors@4.1.1": {},
        "browser-stdout@1.3.1": {},
        "chokidar@3.5.3": {
            "anymatch@3.1.3": {
                "normalize-path@3.0.0": "deduped",
                "picomatch@2.3.1": {}
            },
            ...,
            "normalize-path@3.0.0": {},
            "readdirp@3.6.0": {
                "picomatch@2.3.1": {}
            }
        },
        ...
    }
}
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.

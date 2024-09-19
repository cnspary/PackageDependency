# Pip Dependency Resolver

## Usage

To run the resolver, use the following command with the module path and version as arguments:

```bash
python3 main.py requests # latest version
python3 main.py requests==1.1.0 # specific version
```

The program will:

1. Connect to Postgresql.
2. Fetch the artifact's dependencies.
3. Build a dependency graph and stort the dependency graph to database.

## Example Output

****************************Dependency Graph Node**************************

Graph Node: {'urllib3': '1.26.14', 'certifi': '2022.12.7', 'charset-normalizer': '3.1.0', 'idna': '3.4', None: ''}

****************************Dependency Graph Edge**************************

Graph Edge: {None: {'urllib3', 'idna', 'certifi', 'charset-normalizer'}, 'charset-normalizer': set(), 'idna': set(), 'urllib3': set(), 'certifi': set()}

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.
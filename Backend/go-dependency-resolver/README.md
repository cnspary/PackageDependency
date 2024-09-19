# Go Dependency Resolver

## Usage

To run the crawler, use the following command with the module path and version as arguments:

```bash
go run main.go <module_path> <module_version>
```

For example:

```bash
go run main.go github.com/some/module v1.0.0
```

The program will:

1. Connect to MongoDB.
2. Fetch the module's dependencies.
3. Build a dependency graph and print the build list.

## Example Output

If you run the crawler with a valid module, you should see output like this:

```
github.com/some/module
github.com/some/dependency v1.0.0 => github.com/another/dependency v2.0.0
...
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.

---

This `README` covers project setup, MongoDB configuration, running the application, and basic usage instructions.
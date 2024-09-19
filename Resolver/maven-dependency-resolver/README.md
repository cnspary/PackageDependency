# Maven Dependency Resolver

## Usage

To run the resolver, use the following command with the module path and version as arguments:

```bash
java -jar build/MavenEcoSystem.jar --deptree
```

The program will:

1. Connect to Postgresql.
2. Fetch the artifact's dependencies.
3. Build a dependency graph and stort the dependency graph to database.

## Example Output

If you see the resolving results in table `ArtifactMeta`.`deptree` of database.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.

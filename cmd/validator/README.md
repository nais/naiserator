# Validator

## Syntax
```
make validator
bin/validator --input nais.yaml
```

## Exit codes
| 0 | input file has correct format according to [spec](https://doc.nais.io/nais-application/manifest) |
| 1 | error when invoking validator, e.g. missing or invalid parameters |
| 2 | error when reading input file |
| 3 | parse error when reading JSON or YAML from input file |
| 4 | logic error in input file |

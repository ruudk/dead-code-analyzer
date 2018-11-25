# Dead Code Analyzer Server

## Start the server
```
go run main.go
```

Go to http://localhost:8080/reset and enter all Fully-Qualified Class Names (FQCN) you want to analyze.

You can use this handy snippet to get all FQCN from your `src` directory:
```
find src -name '*.php' | sed 's/.php//g' | sed 's@src/@@g' | sed "s@/@\\\@g"
```

Then [configure the PHP library](https://github.com/ruudk/dead-code-analyzer) and point it to this server's ip.
 
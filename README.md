# jm

Query json lines using [jmespath](https://jmespath.org/).


## usage examples

```bash
# query json from local file
jm <jmespath> <<_EOF
{"key": "value1"}
{"key": "value2"}
_EOF

# query json from remotes. It GETs json from remote in parallel hence the output order is undeterministic.
jm <jmespath> <<_EOF
http://remote1/path1
http://remote1/path2
http://remote2/path1
_EOF
```

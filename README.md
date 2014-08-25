goservestars
============

Go app to serve star info from a database over HTTP+JSON+REST.

Currently serves static star info from a given HYG postgresql database over HTTP/REST at a given port at /star/{id}

For example

```bash
goservestars -database hyg -user jimbob -password nascarrulez -port 8081
```

will start serving. Then, directing a web browser to 

```http
http://localhost:8081/star/24378
```

will return

```json
{id: 24378, name: "Rigel", x: 46.220870971679688, y: 229.943740844726562, z: -33.805030822753906, color: -0.029999999329448, magnitude: 0.180000007152557}
```

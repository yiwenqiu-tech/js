docker build -t js:{version} .

docker logs --tail 50 --follow --timestamps {id}

docker run -d --env-file ./envfile --name js -p 8080:8080 js:0.0.1
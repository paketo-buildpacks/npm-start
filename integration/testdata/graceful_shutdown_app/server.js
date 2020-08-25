const http = require('http');
const leftpad = require('leftpad');

const port = process.env.PORT || 8080;

const server = http.createServer((request, response) => {
  response.end("hello world")
});

process.once('SIGTERM', function (code) {
      console.log('echo from SIGTERM handler');
      server.close();
});

server.listen(port, (err) => {
  if (err) {
    return console.log('something bad happened', err);
  }

  console.log(`NOT vendored server is listening on ${port}`);
});

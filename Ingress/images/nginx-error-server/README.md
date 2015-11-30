# nginx error server

[404-server](https://github.com/kubernetes/contrib/tree/master/404-server) image that provides the additional route `/error` to allow custom error pages

# Description:

The route `/error` expects two arguments: `code` and `format`.
- `code` defines the wich error code is expected to be returned
- `format` the format that should be returned
For instance `/error?code=504&format=json` or `/error?code=502&format=html`

The content to be returned must be a file located inside the directory `/var/www/html`. Following the previous example, is expected a file named 504.json or 502.html inside `/var/www/html`.
If there is no such file it will try to return the content of the most generic code for the error, ie 5xx for any error bigger than 499 for instance.

# How to use it:

- Create a custom docker image using `FROM aledbf/nginx-error-server` as parent adding the custom content inside `/var/www/html`
- Use `aledbf/nginx-error-server` image with a custom volume pointing to `/var/www/html`.

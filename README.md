# why
[![Documentation](https://godoc.org/github.com/BigJk/why?status.svg)](http://godoc.org/github.com/BigJk/why)

One night after having to write some PHP I was asking myself if it was possible to replicate the style of PHP with GO. This was the moment the idea for why was born. I wanted to create a PHP-Like server and scripting language that is extendable through GO and can be used to create websites. This is merely a fun project and shouldn’t be taken too seriously.

## Scripting Language

Instead of writing my own scripting language from scratch I decided to use [Tengo](https://github.com/d5/tengo) as a base. Tengo is easiliy extendable and the syntax is not too far from GO. I use the unstable ``tengo2`` branch because I needed the possibility to call tengo functions from go and not just the other way around. This is important so you can build more advanced extensions.

## Example

#### 1. Download the latest Release (or build the Server)

Download the latest release for your platform from the [Releases](https://github.com/BigJk/why/releases) page (or get the package and build the ``main.go`` in ``cmd/``)

#### 2. Create a ``config.json``

In the directory where the binary of the server is located create the following config:

```
{
  "PublicDir": "./public",
  "BindAddress": ":8765"
}
```

#### 3. Create the ``/public`` directory

This directory will contain your scripts and static files.

#### 4. Create ``/public/index.tengo``

This will be the example script that we want to run. Let's imagine we want to print all the GET parameter and their values into our webpage. Paste the following script into the created file.

```
<!DOCTYPE html>
<html lang="en">
  <title>GET Example</title>
  <body>

    <!?
    
      http.write("Get Parameter:<br>")
      keys := http.GET.keys()
      for i := 0; i < len(keys); i++ {
          http.write("name=", keys[i], ", value=", http.GET.param(keys[i]), "<br>")
      }

    ?!>
    
  </body>
</html>
```

#### 5. Start the Server

Start ``./why`` (or ``./cmd`` if you built it yourself) and visit ``http://127.0.0.1:8765/index?param_1=test&param_2=another_test``. You should see the following in your Browser:
```
Get Parameter:
name=param_1, value=test
name=param_2, value=another_test
```

## Default Variables & Functions

- ``http.method``: Contains the http method of the current request (e.g. ``POST``, ``GET``...).
- ``http.full_uri``: Contains the full url of the current request.
- ``http.path``: Contains only the path of the current request.
- ``http.host``: Contains the hostname or hostname:port of the current request.
- ``http.ip``: Contains the IP of the client that made the current request.
- ``http.proto``: HTTP protocol version of the current request.
- ``http.status_code(<int>)``: This will set the status code of the response.
- ``http.write(...)``: Variadic function that will write into the document. This is like ``echo`` in php.
- ``http.overwrite(...)``: Variadic function that will overwrite all content that was written to the document before.
- ``http.escape(<string>)``: Escapes the string. Can be used to avoid XSS.
- ``http.body()``: Will return the raw post body data.
- ``http.die()``: Will halt the execution of the script and finish the request.

#### GET

- ``http.GET.keys()``: Returns a list of all the present ``GET`` parameters.
- ``http.GET.param(<string>)``: Returns the value of a ``GET`` parameter.

#### POST

- ``http.POST.keys()``: Returns a list of all the present ``POST`` parameters.
- ``http.POST.param(<string>)``: Returns the value of a ``POST`` parameter.

#### HEADER

- ``http.HEADER.keys()``: Returns a list of all the present request headers.
- ``http.HEADER.param(<string>)``: Returns the value of a header entry.
- ``http.HEADER.set(<string>, <string>)``: Set's a response header.

#### COOKIES

- ``http.COOKIES.all()``: Returns all the cookies.
- ``http.COOKIES.param(<string>)``: Returns a cookie by key.
- ``http.COOKIES.set(<object>)``: Set's a cookie.

## Extensibility

Adding custom variables and functions to the scripting engine can be done via the ``Extension`` interface. With the help of Extensions it's possible to add adapters for Databases and various other things.

## Available Extensions

- ``bbolt`` ([docs](https://godoc.org/github.com/BigJk/why/extensions/bbolt)): Key-Value Storage.
- ``jwt`` ([docs](https://godoc.org/github.com/BigJk/why/extensions/jwt)): Generate JWT's and validate and extract data from them.

## More Examples?

Take a look at the ``examples/`` folder. I will keep adding simple usage examples there to illustrate the functionality of why.

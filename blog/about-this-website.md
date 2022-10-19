--
Title: About this website
Date: 18/10/2022
--

# About this website

Finally, I was able to get a good domain name.
This one was unavailable before, and the one I had was too long, which I didn't like.
It was the catalyst for me to start this website.
I made some attempts before, with [elm](https://elm-lang.org/) and also
[react](https://reactjs.org/).
Elm is an enjoyable language, the syntax is comfy (if you enjoy functional
languages), the compiler is helpful, it is simple, and has some interesting
concepts behind it.
However, the curtain unveils when you try to do more complex things and
interoperating with javascript.
In a way, elm treats its users like children, and forces them into a playground,
were they can only [play with things considered good and
correct](https://lukeplant.me.uk/blog/posts/why-im-leaving-elm/#why-don-t-you-just-ing).
React is fine, it does its job pretty well, it has a functional feel, the way it
takes care of html is, in my opinion, easier to use than templating.
It don't remember why I scrapped the website I tried to developed in it, I guess
I didn't enjoy the feel of it.
So this time, instead of generating static pages, I am just writing an entire
backend to serve these pages too.
This was in big part inspired by
[this article](https://xeiaso.net/blog/new-language-blog-backend-2022-03-02).
I was already interested in golang, and  already used it to implement a simple
command line tool.
However, I had yet to put in use its fearless concurrency™ features.
Unfortunatly (or fortunatly) I haven't needed to use these features yet, since
the language has good support for http (and https) out of the box.

## The implementation

So, I started from the beginning, showing a `"Hello World"` message on `/`.
This is pretty simple in go using [net/http](https://pkg.go.dev/net/http).
You can simply register a pattern to be handled by a function.
The handling function will receive two arguments:
- a `ResponseWriter`, which as the name indicates, is where you're supposed to
write your response;
- a `Request` which you can use to extract metadata about the request.

After that you simply use `ListenAndServe` and it will automagically handle
requests.
```go
http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "Hello World")
})

log.Fatal(http.ListenAndServe(":8080", nil))
```

The component that handles the routing of a pattern to the correct handler
function is referred to as a `Mux` (multiplexer).
The default `ServeMux` is interesting, when the pattern ends with `/` it will
actually also capture everything below that path.
For example, for the previous code, entering the path `/foo` would also show the
same message as `/`.
This means that if I want to make a 404 message appear instead of 'hello world'
in the previous example, I have to do it manually in the handler function.
Something similar to this.

```go
http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
```

This feels kinda dirty, probably because I am having multiple "sources of truth" with the `"/".
I also am not sure if using `r.URL.Path` is the best way to do this verification.
Anyway this behaviour was strange to me at first, but I think I understood why it
was designed this way.
With such a simple API, if the behaviour was different, I don't see a way to be
able to handle arbitrary paths, not known beforehand.

Moving on, I started writing html templates for the root page, based on
[html/template](https://pkg.go.dev/html/template).
The usage is pretty simple.
First, load the templates using `template.ParseGlob("templates/*.go.html")`,
which will load all files matching the pattern.
And, when necessary, obtain the resulting html with the `ExecuteTemplate`, which
receives the template name and, optionally, a map of variables.
The name of the template is simply the filename (not the entire filepath), this
makes it kinda awkward in terms of organization.
Currently I have every template in the same folder, but in the future, I want to
check if there is a way I can change the names to reflect their full relative
path, not just the filename.

Serving static files was pretty simple, since golang provides a function that does just that.
```go
http.ServeFile(w, r, "."+r.URL.Path)
```
I am not sure if using `r.URL.Path` this way has vulnerabilities.
However, from what I read in the documentation, the Mux sanitizes this url, and
doesn't allow funny business like doing `/../`.
Either way, I am likely to change this in the future, to allow changing the path
for the static files.

The final thing at the moment of writing is, well... this writing itself, pretty meta.
It is actually written in markdown, and the translation is done with
[goldmark](github.com/yuin/goldmark).
Following the documentation was pretty simple, just point to the markdown data
and obtain the resulting HTML.
To have some pretty highlighting in the `code` tags I used [highlightjs](https://highlightjs.org/).
For now, to serve the blog I simply go through the files in the `blog/` folder
and read their name and tag.
At this moment this is the only file, so it is not a problem.
In the future I want to make this better, maybe by adding a cache to the Mux,
and updating it when the markdown is updated.
I have never used [pool](https://man7.org/linux/man-pages/man2/poll.2.html) so
this might be a good way to learn how it works.

## Making it available to the world

As stated before, the start to this was me getting a new domain.
At first I thought of hosting it in a machine of my own.
However, I don't currently have a low energy machine, like a raspberry.
Furthermore, after searching a bit, I cannot just alter the settings on my
router since the ISP does not allow that.
And, I do not have a static IP, so I would have to setup some kind of dynamic
dns service.
So I ended up just looking for a cheap VPS.
The search led me to Digital Ocean
[droplets](https://www.digitalocean.com/pricing/droplets#basic-droplets), the
cheapest one is $4.00/mo and the specs are good enough.
Even better, I still had a coupon from university, so for a few months I will
have it for free.

To setup https I simply used [certbot](https://certbot.eff.org/) and followed
the instructions to generate the public and private keys.

I ran into a slight problem building the executable for the server.
The go compiler was using the entire RAM to build one of the packages imported.
The solution to this was simply to create a swap file and activate it.
I think this was the first time I actually required swap to achieve something.

I expect compiling the code on the droplet will be temporary.
I already generate a docker using [nix](https://nixos.org/), the image ends up
very minimal, about 50MB, although I think this can be reduced further.
The image works well, the only reason I am not using it is because I am still
figuring out the best way to allow configuration.
Currently by default, the server listens at port 8080 without TLS, in the
droplet, I obviously want to alter this.

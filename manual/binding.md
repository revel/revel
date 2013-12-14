---
title: Binding Parameters
layout: manual
---

Revel tries to make the conversion of parameters into their desired Go types as
easy as possible.  This conversion from string to another type is referred to as
"data binding".

## Params

All request parameters are collected into a single `Params` object.  That includes:
* URL Path parameters
* URL Query parameters
* Form values (Multipart or not)
* File uploads

This is the definition ([godoc](../docs/godoc/params.html)):

<pre class="prettyprint lang-go">
type Params struct {
	url.Values
	Files map[string][]*multipart.FileHeader
}
</pre>

The embedded `url.Values` ([godoc](http://www.golang.org/pkg/net/url/#Values))
does provide accessors for simple values, but developers will find it easier to
use Revel's data-binding mechanisms for any non-string values.

## Action arguments

Parameters may be accepted directly as method arguments by the action.  For
example:

<pre class="prettyprint lang-go">
func (c AppController) Action(name string, ids []int, user User, img []byte) revel.Result {
	...
}
</pre>

Before invoking the action, Revel asks its Binder to convert parameters of those
names to the requested data type.  If the binding is unsuccessful for any
reason, the parameter will have the zero value for its type.

## Binder

To bind a parameter to a data type, use Revel's Binder
([godoc](../docs/godoc/binder.html)).  It is integrated with the Params object
as the following example shows:

{% raw %}
<pre class="prettyprint lang-go">
func (c SomeController) Action() revel.Result {
	var ids []int
	c.Params.Bind(&amp;ids, "ids")
	...
}
</pre>
{% endraw %}

The following data types are supported out of the box:
* Ints of all widths
* Bools
* Pointers to any supported type
* Slices of any supported type
* Structs
* time.Time for dates and times
* \*os.File, \[\]byte, io.Reader, io.ReadSeeker for file uploads

The following sections describe the syntax for these types.  It is also useful
to refer to [the source code](../docs/src/binder.html) if more detail is required.

### Booleans

The string values "true", "on", and "1" are all treated as **true**.  Else, the
bound value will be **false**.

### Slices

There are two supported syntaxes for binding slices: ordered or unordered.

Ordered:

	?ids[0]=1
	&ids[1]=2
	&ids[3]=4

Results in the slice `[]int{1, 2, 0, 4}`

Unordered:

	?ids[]=1
	&ids[]=2
	&ids[]=3

results in the slice `[]int{1, 2, 3}`

**Note:** Only ordered slices should be used when binding a slice of structs:

	?user[0].Id=1
	&user[0].Name=rob
	&user[1].Id=2
	&user[1].Name=jenny

### Structs

Structs are bound using a simple dot notation:

	?user.Id=1
	&user.Name=rob
	&user.Friends[]=2
	&user.Friends[]=3
	&user.Father.Id=5
	&user.Father.Name=Hermes

would bind a structure defined as:

<pre class="prettyprint lang-go">
type User struct {
	Id int
	Name string
	Friends []int
	Father User
}
</pre>

**Note:** Properties must be exported in order to be bound.

### Date / Time

The SQL standard time formats \["2006-01-02", "2006-01-02 15:04"\] are built in.

More may be added by the application, using
[the official pattern](http://golang.org/pkg/time/#pkg-constants).  Simply add
the pattern to recognize to the `TimeFormats` variable, like this:

<pre class="prettyprint lang-go">
func init() {
	revel.TimeFormats = append(revel.TimeFormats, "01/02/2006")
}
</pre>

### File Uploads

File uploads may be bound to any of the following types:
* \*os.File
* \[\]byte
* io.Reader
* io.ReadSeeker

This is a wrapper around the upload handling provided by
[Go's multipart package](http://golang.org/pkg/mime/multipart/).  The bytes
stay in memory unless they exceed a threshold (10MB by default), in which case
they are written to a temp file.

**Note:** Binding a file upload to `os.File` requires Revel to write it to a
temp file (if it wasn't already), making it less efficient than the other types.

### Custom Binders

The application may define its own binders to take advantage of this framework.

It need only implement the [binder interface](../docs/godoc/binder.html#Binder) and register the type for which it
should be called:

<pre class="prettyprint lang-go">
var myBinder = revel.Binder{
	Bind: func(params *revel.Params, name string, typ reflect.Type) reflect.Value {...},
	Unbind: func(output map[string]string, name string, val interface{}) {...},
}

func init() {
	revel.TypeBinders[reflect.TypeOf(MyType{})] = myBinder
}
</pre>

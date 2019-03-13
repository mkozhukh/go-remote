Remote - client/server transport for js/go
======================

## Server side

```go
s := remote.NewServer()
guard := func(req *http.Request) bool {
    return login.CheckAccess(req, auth.AdminAccess)
}

s.Register("snippet", &SnippetAPI{})
s.RegisterWithGuard("admin", &SnippetAdminAPI{}, guard)
s.RegisterConstant("versions", "1.0")


s.RegisterProvider(func(r *http.Request) *User {
	return User{}
})
s.RegisterVariable("user", &User{})

router.Handle("/api/v1", s)
```

## Client side

```html
<script type="text/javascript" src="https://snippet.webix.com/api/v1"></script>
<script>
  alert(remote.data.version);
  remote.api.snippet.Save(config);

  remote.onload = function(promise){
  	//called each time when server side communcation started
  };

  remote.onerror = function(err){
  	//called each time when server side error occurs
  };
</script>
```


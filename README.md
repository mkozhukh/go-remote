Remote - JS RPC for Go
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
  
router.Handle("/api/v1", s)
```

## Client side

```html
<script type="text/javascript" src="https://snippet.webix.com/api/v1"></script>
<script>
  alert(remote.data.version);
  remote.api.snippet.Save(config);
</script>
```


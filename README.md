# grouper
A Generic version of golang.org/x/sync/errgroup
This version doesn't allow reuse or unlimite calls to a Go functions, but in exchange the Wait methods returns a slice of all of the successful function returns.
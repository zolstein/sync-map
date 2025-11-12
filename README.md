[![Go Reference](https://pkg.go.dev/badge/github.com/zolstein/sync-map.svg)](https://pkg.go.dev/github.com/zolstein/sync-map)

# sync-map

A fork of Go's `sync.Map`, but with generics.

## Why...

### ... not just use `sync.Map`?

Declaring the types of variables in your code helps document their usage and prevent programming errors.
It's unfortunate that `sync.Map` predates generics - now that we have them I think even people who want
to use a concurrent map should have access to type checking.

### ... not just create a wrapper around `sync.Map` to provide type-safety?

Putting value behind a pointer to the heap has a cost - probably a bigger one than you expect.
If you're working with values of mixed types, this may be necessary. If you only want to work with values of
one type, it just adds overhead. Modifying the map to support generics natively allows us to get 
(sometimes substantially) better performance by using simple types as keys.

### ... not use my favorite concurrent map implementation?

`sync.Map` is not the fastest possible implementation of a concurrent map. However, it is relatively simple and its
performance characteristics are well-understood and optimized for common use-cases. In the past, I have tested
packages advertising a faster concurrent map and struggled to get the performance to equal `sync.Map`. 

This may be caused in part by the opaque way in which Go handles hashing - the Go runtime defines its own hash function
in the runtime for its internal maps, but other map implementations require a user-defined hash function. Historically,
Go has not provided a good, easy way to implement this. Recent changes to the `maphash` package may have improved this.

Regardless of the reason, though, it makes sense to provide the same algorithm but with support for generics to ensure
the performance characteristics are not worse than the default option.

### ... are `CompareAndSwap` and `CompareAndDelete` functions, not methods?

`CompareAndSwap` and `CompareAndDelete` are useful operations to support on a concurrent map. These operations require
the values in the map to be `comparable` - otherwise, how will you compare them to know if you should swap or delete the
values? However, we also want to be able to have `Map` store non-comparable values.

The compromise is to remove these methods from the regular `Map` type, and create functions that can apply a tighter
bound on the value type.

### ... not use the new `sync.Map` implementation from Go 1.24?

In 1.24, Go updated the implementation of `sync.Map` to use a concurrent hash-trie. The underlying interal `HashTrieMap`
already supports generics, it's just not exported in a way that makes the generic version available. I would like
to provide an exposed version of this. However, the `HashTrieMap` implementation uses `internal/abi` functionality,
which isn't accessible or safe to re-implement outside the standard library, so I don't know that there's a good way to
do this. Hopefully `sync/v2` with an officially-supported generic map is coming soon.

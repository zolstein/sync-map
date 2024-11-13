[![Go Reference](https://pkg.go.dev/badge/github.com/zolstein/sync-map.svg)](https://pkg.go.dev/github.com/zolstein/sync-map)

# sync-map

A fork of Go's `sync.Map`, but with generics.

## Why...

### ... not just use `sync.Map`?

Declaring the types of variables in your code helps document their usage and prevent programming errors.
It's unfortunate that `sync.Map` predates generics, but now that we have them I don't think there's a good
argument that anyone who wants to use a concurrent map should have to lose access to type checking.

### ... not just create a wrapper around `sync.Map` to provide type-safety?

Putting value behind a pointer to the heap has a cost. (Probably a bigger one than you expect.) 
If you're working with values of mixed types, this may be necessary. If you only want to work with values of
one type, it's pointless overhead. Modifying the map to support generics natively allows us to get better performance,
sometimes substantially so, using simple types as keys.

### ... not use my favorite concurrent map implementation?

`sync.Map` is not the fastest possible implementation of a concurrent map. However, its implementation is relatively
simple and its performance characteristics are well-understood. In the past, when I have tried to use modules
advertising a faster concurrent map, I have struggled to get equal performance than `sync.Map`. This may be caused in
part by Go's lack of a good, user-accessible way to hash an arbitrary comparable object. Regardless of the reason, it
seems worth providing the same algorithm but with support for generics to ensure the performance characteristics are
not worse than the default option.

### ... did you create a separate `CasMap`?

`CompareAndSwap` and `CompareAndDelete` are useful operations to support on a concurrent map. However, these operations
require the values in the map to be `comparable` - otherwise, how will you compare them to know if you should swap or
delete the values? However, we also want to be able to have `Map` store non-comparable values.

The compromise is to remove these functions from the regular `Map` type, then create a separate map type with a tighter
bound on its value type that can support them.

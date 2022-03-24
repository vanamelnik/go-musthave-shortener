/*

Staticlint is a tool for static analysis of Go programs.


Usage:
        staticlint [-flag] [package]

Available checks:

		-asmdecl:
			report mismatches between assembly files and Go declarations

		-assign:
			check for useless assignments
			This checker reports assignments of the form x = x or a[i] = a[i].
			These are almost always useless, and even when they aren't they are
			usually a mistake.

		-atomic:
			check for common mistakes using the sync/atomic package
			The atomic checker looks for assignment statements of the form:
				x = atomic.AddUint64(&x, 1)
			which are not atomic.

		-bools:
			check for common mistakes involving boolean operators

		-buildssa:
			build SSA-form IR for later passes

		-buildtag:
			check that +build tags are well-formed and correctly located

		-cgocall:
			detect some violations of the cgo pointer passing rules
			Check for invalid cgo pointer passing.
			This looks for code that uses cgo to call C code passing values
			whose types are almost always invalid according to the cgo pointer
			sharing rules.
			Specifically, it warns about attempts to pass a Go chan, map, func,
			or slice to C, either directly, or via a pointer, array, or struct.

		-composites:
			check for unkeyed composite literals
			This analyzer reports a diagnostic for composite literals of struct
			types imported from another package that do not use the field-keyed
			syntax. Such literals are fragile because the addition of a new field
			(even if unexported) to the struct will cause compilation to fail.
			As an example,
				err = &net.DNSConfigError{err}
			should be replaced by:
				err = &net.DNSConfigError{Err: err}


		-copylocks:
			check for locks erroneously passed by value
			Inadvertently copying a value containing a lock, such as sync.Mutex or
			sync.WaitGroup, may cause both copies to malfunction. Generally such
			values should be referred to through a pointer.

		-ctrlflow:
			build a control-flow graph

		-deepequalerrors:
			check for calls of reflect.DeepEqual on error values
			The deepequalerrors checker looks for calls of the form:
			    reflect.DeepEqual(err1, err2)
			where err1 and err2 are errors. Using reflect.DeepEqual to compare
			errors is discouraged.

		-errorsas:
			report passing non-pointer or non-error values to errors.As
			The errorsas analysis reports calls to errors.As where the type
			of the second argument is not a pointer to a type implementing error.

		-findcall:
			find calls to a particular function
			The findcall analysis reports calls to functions or methods
			of a particular name.

		-framepointer:
			report assembly that clobbers the frame pointer before saving it

		-httpresponse:
			check for mistakes using HTTP responses
			A common mistake when using the net/http package is to defer a function
			call to close the http.Response Body before checking the error that
			determines whether the response is valid:
				resp, err := http.Head(url)
				defer resp.Body.Close()
				if err != nil {
					log.Fatal(err)
				}
				// (defer statement belongs here)
			This checker helps uncover latent nil dereference bugs by reporting a
			diagnostic for such mistakes.

		-ifaceassert:
			detect impossible interface-to-interface type assertions
			This checker flags type assertions v.(T) and corresponding type-switch cases
			in which the static type V of v is an interface that cannot possibly implement
			the target interface T. This occurs when V and T contain methods with the same
			name but different signatures. Example:
				var v interface {
					Read()
				}
				_ = v.(io.Reader)
			The Read method in v has a different signature than the Read method in
			io.Reader, so this assertion cannot succeed.


		-inspect:
			optimize AST traversal for later passes

		-loopclosure:
			check references to loop variables from within nested functions
			This analyzer checks for references to loop variables from within a
			function literal inside the loop body. It checks only instances where
			the function literal is called in a defer or go statement that is the
			last statement in the loop body, as otherwise we would need whole
			program analysis.
			For example:
				for i, v := range s {
					go func() {
						println(i, v) // not what you might expect
					}()
				}
			See: https://golang.org/doc/go_faq.html#closures_and_goroutines

		-lostcancel:
			check cancel func returned by context.WithCancel is called
			The cancellation function returned by context.WithCancel, WithTimeout,
			and WithDeadline must be called or the new context will remain live
			until its parent context is cancelled.
			(The background context is never cancelled.)

		-nilfunc:
			check for useless comparisons between functions and nil
			A useless comparison is one like f == nil as opposed to f() == nil.

		-nilness:
			check for redundant or impossible nil comparisons
			The nilness checker inspects the control-flow graph of each function in
			a package and reports nil pointer dereferences, degenerate nil
			pointers, and panics with nil values. A degenerate comparison is of the form
			x==nil or x!=nil where x is statically known to be nil or non-nil. These are
			often a mistake, especially in control flow related to errors. Panics with nil
			values are checked because they are not detectable by
				if r := recover(); r != nil {
			This check reports conditions such as:
				if f == nil { // impossible condition (f is a function)
				}
			and:
				p := &v
				...
				if p != nil { // tautological condition
				}
			and:
				if p == nil {
					print(*p) // nil dereference
				}
			and:
				if p == nil {
					panic(p)
				}


		-pkgfact:
			gather name/value pairs from constant declarations

		-printf:
			check consistency of Printf format strings and arguments
			The check applies to known functions (for example, those in package fmt)
			as well as any detected wrappers of known functions.
			A function that wants to avail itself of printf checking but is not
			found by this analyzer's heuristics (for example, due to use of
			dynamic calls) can insert a bogus call:
				if false {
					_ = fmt.Sprintf(format, args...) // enable printf checking
				}
			The -funcs flag specifies a comma-separated list of names of additional
			known formatting functions or methods. If the name contains a period,
			it must denote a specific function using one of the following forms:
				dir/pkg.Function
				dir/pkg.Type.Method
				(*dir/pkg.Type).Method
			Otherwise the name is interpreted as a case-insensitive unqualified
			identifier such as "errorf". Either way, if a listed name ends in f, the
			function is assumed to be Printf-like, taking a format string before the
			argument list. Otherwise it is assumed to be Print-like, taking a list
			of arguments with no format string.


		-reflectvaluecompare:
			check for comparing reflect.Value values with == or reflect.DeepEqual
			The reflectvaluecompare checker looks for expressions of the form:
			    v1 == v2
			    v1 != v2
			    reflect.DeepEqual(v1, v2)
			where v1 or v2 are reflect.Values. Comparing reflect.Values directly
			is almost certainly not correct, as it compares the reflect package's
			internal representation, not the underlying value.
			Likely what is intended is:
			    v1.Interface() == v2.Interface()
			    v1.Interface() != v2.Interface()
			    reflect.DeepEqual(v1.Interface(), v2.Interface())


		-shadow:
			check for possible unintended shadowing of variables
			This analyzer check for shadowed variables.
			A shadowed variable is a variable declared in an inner scope
			with the same name and type as a variable in an outer scope,
			and where the outer variable is mentioned after the inner one
			is declared.
			(This definition can be refined; the module generates too many
			false positives and is not yet enabled by default.)
			For example:
				func BadRead(f *os.File, buf []byte) error {
					var err error
					for {
						n, err := f.Read(buf) // shadows the function variable 'err'
						if err != nil {
							break // causes return of wrong value
						}
						foo(buf)
					}
					return err
				}


		-shift:
			check for shifts that equal or exceed the width of the integer

		-sigchanyzer:
			check for unbuffered channel of os.Signal
			This checker reports call expression of the form signal.Notify(c <-chan os.Signal, sig ...os.Signal),
			where c is an unbuffered channel, which can be at risk of missing the signal.

		-sortslice:
			check the argument type of sort.Slice
			sort.Slice requires an argument of a slice type. Check that
			the interface{} value passed to sort.Slice is actually a slice.

		-stdmethods:
			check signature of methods of well-known interfaces
			Sometimes a type may be intended to satisfy an interface but may fail to
			do so because of a mistake in its method signature.
			For example, the result of this WriteTo method should be (int64, error),
			not error, to satisfy io.WriterTo:
				type myWriterTo struct{...}
			        func (myWriterTo) WriteTo(w io.Writer) error { ... }
			This check ensures that each method whose name matches one of several
			well-known interface methods from the standard library has the correct
			signature for that interface.
			Checked method names include:
				Format GobEncode GobDecode MarshalJSON MarshalXML
				Peek ReadByte ReadFrom ReadRune Scan Seek
				UnmarshalJSON UnreadByte UnreadRune WriteByte
				WriteTo


		-stringintconv:
			check for string(int) conversions
			This checker flags conversions of the form string(x) where x is an integer
			(but not byte or rune) type. Such conversions are discouraged because they
			return the UTF-8 representation of the Unicode code point x, and not a decimal
			string representation of x as one might expect. Furthermore, if x denotes an
			invalid code point, the conversion cannot be statically rejected.
			For conversions that intend on using the code point, consider replacing them
			with string(rune(x)). Otherwise, strconv.Itoa and its equivalents return the
			string representation of the value in the desired base.


		-structtag:
			check that struct field tags conform to reflect.StructTag.Get
			Also report certain struct tags (json, xml) used with unexported fields.

		-testinggoroutine:
			report calls to (*testing.T).Fatal from goroutines started by a test.
			Functions that abruptly terminate a test, such as the Fatal, Fatalf, FailNow, and
			Skip{,f,Now} methods of *testing.T, must be called from the test goroutine itself.
			This checker detects calls to these functions that occur within a goroutine
			started by the test. For example:
			func TestFoo(t *testing.T) {
			    go func() {
			        t.Fatal("oops") // error: (*T).Fatal called from non-test goroutine
			    }()
			}


		-tests:
			check for common mistaken usages of tests and examples
			The tests checker walks Test, Benchmark and Example functions checking
			malformed names, wrong signatures and examples documenting non-existent
			identifiers.
			Please see the documentation for package testing in golang.org/pkg/testing
			for the conventions that are enforced for Tests, Benchmarks, and Examples.

		-unmarshal:
			report passing non-pointer or non-interface values to unmarshal
			The unmarshal analysis reports calls to functions such as json.Unmarshal
			in which the argument type is not a pointer or an interface.

		-unreachable:
			check for unreachable code
			The unreachable analyzer finds statements that execution can never reach
			because they are preceded by an return statement, a call to panic, an
			infinite loop, or similar constructs.

		-unsafeptr:
			check for invalid conversions of uintptr to unsafe.Pointer
			The unsafeptr analyzer reports likely incorrect uses of unsafe.Pointer
			to convert integers to pointers. A conversion from uintptr to
			unsafe.Pointer is invalid if it implies that there is a uintptr-typed
			word in memory that holds a pointer value, because that word will be
			invisible to stack copying and to the garbage collector.

		-unusedresult:
			check for unused results of calls to some functions
			Some functions like fmt.Errorf return a result and have no side effects,
			so it is always a mistake to discard the result. This analyzer reports
			calls to certain functions in which the result of the call is ignored.
			The set of functions may be controlled using flags.

		-unusedwrite:
			checks for unused writes
			The analyzer reports instances of writes to struct fields and
			arrays that are never read. Specifically, when a struct object
			or an array is copied, its elements are copied implicitly by
			the compiler, and any element write to this copy does nothing
			with the original object.
			For example:
				type T struct { x int }
				func f(input []T) {
					for i, v := range input {  // v is a copy
						v.x = i  // unused write to field x
					}
				}
			Another example is about non-pointer receiver:
				type T struct { x int }
				func (t T) f() {  // t is a copy
					t.x = i  // unused write to field x
				}


		-usesgenerics:
			detect whether a package uses generics features
			The usesgenerics analysis reports whether a package directly or transitively
			uses certain features associated with generic programming in Go.

		-osexit:
			check for os.Exit call in main function within package main

		-SA1007:
			Invalid URL in `net/url.Parse`
			Available since
			    2017.1


		-SA1029:
			Inappropriate key in call to `context.WithValue`
			The provided key must be comparable and should not be
			of type string or any other built-in type to avoid collisions between
			packages using context. Users of `WithValue` should define their own
			types for keys.
			To avoid allocating when assigning to an `interface{}`,
			context keys often have concrete type struct{}. Alternatively,
			exported context key variables' static type should be a pointer or
			interface.
			Available since
			    2020.1


		-SA4020:
			Unreachable case clause in a type switch
			In a type switch like the following
			    type T struct{}
			    func (T) Read(b []byte) (int, error) { return 0, nil }
			    var v interface{} = T{}
			    switch v.(type) {
			    case io.Reader:
			        // ...
			    case T:
			        // unreachable
			    }
			the second case clause can never be reached because `T` implements
			`io.Reader` and case clauses are evaluated in source order.
			Another example:
			    type T struct{}
			    func (T) Read(b []byte) (int, error) { return 0, nil }
			    func (T) Close() error { return nil }
			    var v interface{} = T{}
			    switch v.(type) {
			    case io.Reader:
			        // ...
			    case io.ReadCloser:
			        // unreachable
			    }
			Even though `T` has a `Close` method and thus implements `io.ReadCloser`,
			`io.Reader` will always match first. The method set of `io.Reader` is a
			subset of `io.ReadCloser`. Thus it is impossible to match the second
			case without matching the first case.
			Structurally equivalent interfaces
			A special case of the previous example are structurally identical
			interfaces. Given these declarations
			    type T error
			    type V error
			    func doSomething() error {
			        err, ok := doAnotherThing()
			        if ok {
			            return T(err)
			        }
			        return U(err)
			    }
			the following type switch will have an unreachable case clause:
			    switch doSomething().(type) {
			    case T:
			        // ...
			    case V:
			        // unreachable
			    }
			`T` will always match before V because they are structurally equivalent
			and therefore `doSomething()`'s return value implements both.
			Available since
			    2019.2


		-SA4025:
			Integer division of literals that results in zero
			When dividing two integer constants, the result will
			also be an integer. Thus, a division such as '2 / 3' results in '0'.
			This is true for all of the following examples:
				_ = 2 / 3
				const _ = 2 / 3
				const _ float64 = 2 / 3
				_ = float64(2 / 3)
			Staticcheck will flag such divisions if both sides of the division are
			integer literals, as it is highly unlikely that the division was
			intended to truncate to zero. Staticcheck will not flag integer
			division involving named constants, to avoid noisy positives.
			Available since
			    2021.1


		-SA1008:
			Non-canonical key in `http.Header` map
			Keys in `http.Header` maps are canonical, meaning they follow a specific
			combination of uppercase and lowercase letters. Methods such as
			`http.Header.Add` and `http.Header.Del` convert inputs into this canonical
			form before manipulating the map.
			When manipulating `http.Header` maps directly, as opposed to using the
			provided methods, care should be taken to stick to canonical form in
			order to avoid inconsistencies. The following piece of code
			demonstrates one such inconsistency:
			    h := http.Header{}
			    h["etag"] = []string{"1234"}
			    h.Add("etag", "5678")
			    fmt.Println(h)
			    // Output:
			    // map[Etag:[5678] etag:[1234]]
			The easiest way of obtaining the canonical form of a key is to use
			`http.CanonicalHeaderKey`.
			Available since
			    2017.1


		-SA1023:
			Modifying the buffer in an `io.Writer` implementation
			`Write` must not modify the slice data, even temporarily.
			Available since
			    2017.1


		-SA4004:
			The loop exits unconditionally after one iteration
			Available since
			    2017.1


		-SA4005:
			Field assignment that will never be observed. Did you mean to use a pointer receiver?
			Available since
			    2021.1


		-SA9002:
			Using a non-octal `os.FileMode` that looks like it was meant to be in octal.
			Available since
			    2017.1


		-SA1025:
			It is not possible to use `(*time.Timer).Reset`'s return value correctly
			Available since
			    2019.1


		-SA2002:
			Called `testing.T.FailNow` or `SkipNow` in a goroutine, which isn't allowed
			Available since
			    2017.1


		-SA2003:
			Deferred `Lock` right after locking, likely meant to defer `Unlock` instead
			Available since
			    2017.1


		-SA9004:
			Only the first constant has an explicit type
			In a constant declaration such as the following:
			    const (
			        First byte = 1
			        Second     = 2
			    )
			the constant Second does not have the same type as the constant First.
			This construct shouldn't be confused with
			    const (
			        First byte = iota
			        Second
			    )
			where `First` and `Second` do indeed have the same type. The type is only
			passed on when no explicit value is assigned to the constant.
			When declaring enumerations with explicit values it is therefore
			important not to write
			    const (
			          EnumFirst EnumType = 1
			          EnumSecond         = 2
			          EnumThird          = 3
			    )
			This discrepancy in types can cause various confusing behaviors and
			bugs.
			Wrong type in variable declarations
			The most obvious issue with such incorrect enumerations expresses
			itself as a compile error:
			    package pkg
			    const (
			        EnumFirst  uint8 = 1
			        EnumSecond       = 2
			    )
			    func fn(useFirst bool) {
			        x := EnumSecond
			        if useFirst {
			            x = EnumFirst
			        }
			    }
			fails to compile with
			    ./const.go:11:5: cannot use EnumFirst (type uint8) as type int in assignment
			Losing method sets
			A more subtle issue occurs with types that have methods and optional
			interfaces. Consider the following:
			    package main
			    import "fmt"
			    type Enum int
			    func (e Enum) String() string {
			        return "an enum"
			    }
			    const (
			        EnumFirst  Enum = 1
			        EnumSecond      = 2
			    )
			    func main() {
			        fmt.Println(EnumFirst)
			        fmt.Println(EnumSecond)
			    }
			This code will output
			    an enum
			    2
			as `EnumSecond` has no explicit type, and thus defaults to `int`.
			Available since
			    2019.1


		-SA6002:
			Storing non-pointer values in `sync.Pool` allocates memory
			A `sync.Pool` is used to avoid unnecessary allocations and reduce the
			amount of work the garbage collector has to do.
			When passing a value that is not a pointer to a function that accepts
			an interface, the value needs to be placed on the heap, which means an
			additional allocation. Slices are a common thing to put in sync.Pools,
			and they're structs with 3 fields (length, capacity, and a pointer to
			an array). In order to avoid the extra allocation, one should store a
			pointer to the slice instead.
			See the comments on https://go-review.googlesource.com/c/go/+/24371
			that discuss this problem.
			Available since
			    2017.1


		-SA1004:
			Suspiciously small untyped constant in `time.Sleep`
			The `time`.Sleep function takes a `time.Duration` as its only argument.
			Durations are expressed in nanoseconds. Thus, calling `time.Sleep(1)`
			will sleep for 1 nanosecond. This is a common source of bugs, as sleep
			functions in other languages often accept seconds or milliseconds.
			The `time` package provides constants such as `time.Second` to express
			large durations. These can be combined with arithmetic to express
			arbitrary durations, for example `5 * time.Second` for 5 seconds.
			If you truly meant to sleep for a tiny amount of time, use
			`n * time.Nanosecond` to signal to Staticcheck that you did mean to sleep
			for some amount of nanoseconds.
			Available since
			    2017.1


		-SA1011:
			Various methods in the `strings` package expect valid UTF-8, but invalid input is provided
			Available since
			    2017.1


		-SA1026:
			Cannot marshal channels or functions
			Available since
			    2019.2


		-SA4001:
			`&*x` gets simplified to `x`, it does not copy `x`
			Available since
			    2017.1


		-SA4008:
			The variable in the loop condition never changes, are you incrementing the wrong variable?
			Available since
			    2017.1


		-SA4010:
			The result of `append` will never be observed anywhere
			Available since
			    2017.1


		-SA5004:
			`for { select { ...` with an empty default branch spins
			Available since
			    2017.1


		-SA1017:
			Channels used with `os/signal.Notify` should be buffered
			The `os/signal` package uses non-blocking channel sends when delivering
			signals. If the receiving end of the channel isn't ready and the
			channel is either unbuffered or full, the signal will be dropped. To
			avoid missing signals, the channel should be buffered and of the
			appropriate size. For a channel used for notification of just one
			signal value, a buffer of size 1 is sufficient.
			Available since
			    2017.1


		-SA1019:
			Using a deprecated function, variable, constant or field
			Available since
			    2017.1


		-SA4009:
			A function argument is overwritten before its first use
			Available since
			    2017.1


		-SA4022:
			Comparing the address of a variable against nil
			Code such as `if &x == nil` is meaningless, because taking the address of a variable always yields a non-nil pointer.
			Available since
			    2020.1


		-SA5009:
			Invalid Printf call
			Available since
			    2019.2


		-SA9005:
			Trying to marshal a struct with no public fields nor custom marshaling
			The `encoding/json` and `encoding/xml` packages only operate on exported
			fields in structs, not unexported ones. It is usually an error to try
			to (un)marshal structs that only consist of unexported fields.
			This check will not flag calls involving types that define custom
			marshaling behavior, e.g. via `MarshalJSON` methods. It will also not
			flag empty structs.
			Available since
			    2019.2


		-SA4019:
			Multiple, identical build constraints in the same file
			Available since
			    2017.1


		-SA1002:
			Invalid format in `time.Parse`
			Available since
			    2017.1


		-SA2000:
			`sync.WaitGroup.Add` called inside the goroutine, leading to a race condition
			Available since
			    2017.1


		-SA2001:
			Empty critical section, did you mean to defer the unlock?
			Empty critical sections of the kind
			    mu.Lock()
			    mu.Unlock()
			are very often a typo, and the following was intended instead:
			    mu.Lock()
			    defer mu.Unlock()
			Do note that sometimes empty critical sections can be useful, as a
			form of signaling to wait on another goroutine. Many times, there are
			simpler ways of achieving the same effect. When that isn't the case,
			the code should be amply commented to avoid confusion. Combining such
			comments with a `//lint:ignore` directive can be used to suppress this
			rare false positive.
			Available since
			    2017.1


		-SA4000:
			Boolean expression has identical expressions on both sides
			Available since
			    2017.1


		-SA6000:
			Using `regexp.Match` or related in a loop, should use `regexp.Compile`
			Available since
			    2017.1


		-SA1000:
			Invalid regular expression
			Available since
			    2017.1


		-SA1001:
			Invalid template
			Available since
			    2017.1


		-SA1028:
			`sort.Slice` can only be used on slices
			The first argument of sort.Slice must be a slice.
			Available since
			    2020.1


		-SA4011:
			Break statement with no effect. Did you mean to break out of an outer loop?
			Available since
			    2017.1


		-SA4027:
			(*net/url.URL).Query returns a copy, modifying it doesn't change the URL
			`(*net/url.URL).Query` parses the current value of `net/url.URL.RawQuery`
			and returns it as a map of type `net/url.Values`. Subsequent changes to
			this map will not affect the URL unless the map gets encoded and
			assigned to the URL's `RawQuery`.
			As a consequence, the following code pattern is an expensive no-op:
			`u.Query().Add(key, value)`.
			Available since
			    2021.1


		-SA5011:
			Possible nil pointer dereference
			A pointer is being dereferenced unconditionally, while
			also being checked against nil in another place. This suggests that
			the pointer may be nil and dereferencing it may panic. This is
			commonly a result of improperly ordered code or missing return
			statements. Consider the following examples:
			    func fn(x *int) {
			        fmt.Println(*x)
			        // This nil check is equally important for the previous dereference
			        if x != nil {
			            foo(*x)
			        }
			    }
			    func TestFoo(t *testing.T) {
			        x := compute()
			        if x == nil {
			            t.Errorf("nil pointer received")
			        }
			        // t.Errorf does not abort the test, so if x is nil, the next line will panic.
			        foo(*x)
			    }
			Staticcheck tries to deduce which functions abort control flow.
			For example, it is aware that a function will not continue
			execution after a call to `panic` or `log.Fatal`. However, sometimes
			this detection fails, in particular in the presence of
			conditionals. Consider the following example:
			    func Log(msg string, level int) {
			        fmt.Println(msg)
			        if level == levelFatal {
			            os.Exit(1)
			        }
			    }
			    func Fatal(msg string) {
			        Log(msg, levelFatal)
			    }
			    func fn(x *int) {
			        if x == nil {
			            Fatal("unexpected nil pointer")
			        }
			        fmt.Println(*x)
			    }
			Staticcheck will flag the dereference of x, even though it is perfectly
			safe. Staticcheck is not able to deduce that a call to
			Fatal will exit the program. For the time being, the easiest
			workaround is to modify the definition of Fatal like so:
			    func Fatal(msg string) {
			        Log(msg, levelFatal)
			        panic("unreachable")
			    }
			We also hard-code functions from common logging packages such as
			logrus. Please file an issue if we're missing support for a
			popular package.
			Available since
			    2020.1


		-SA1013:
			`io.Seeker.Seek` is being called with the whence constant as the first argument, but it should be the second
			Available since
			    2017.1


		-SA4013:
			Negating a boolean twice (`!!b`) is the same as writing `b`. This is either redundant, or a typo.
			Available since
			    2017.1


		-SA4014:
			An if/else if chain has repeated conditions and no side-effects; if the condition didn't match the first time, it won't match the second time, either
			Available since
			    2017.1


		-SA5005:
			The finalizer references the finalized object, preventing garbage collection
			A finalizer is a function associated with an object that runs when the
			garbage collector is ready to collect said object, that is when the
			object is no longer referenced by anything.
			If the finalizer references the object, however, it will always remain
			as the final reference to that object, preventing the garbage
			collector from collecting the object. The finalizer will never run,
			and the object will never be collected, leading to a memory leak. That
			is why the finalizer should instead use its first argument to operate
			on the object. That way, the number of references can temporarily go
			to zero before the object is being passed to the finalizer.
			Available since
			    2017.1


		-SA5012:
			Passing odd-sized slice to function expecting even size
			Some functions that take slices as parameters expect the slices to have an even number of elements.
			Often, these functions treat elements in a slice as pairs.
			For example, `strings.NewReplacer` takes pairs of old and new strings,
			and calling it with an odd number of elements would be an error.
			Available since
			    2020.2


		-SA9001:
			Defers in range loops may not run when you expect them to
			Available since
			    2017.1


		-SA1003:
			Unsupported argument to functions in `encoding/binary`
			The `encoding/binary` package can only serialize types with known sizes.
			This precludes the use of the `int` and `uint` types, as their sizes
			differ on different architectures. Furthermore, it doesn't support
			serializing maps, channels, strings, or functions.
			Before Go 1.8, `bool` wasn't supported, either.
			Available since
			    2017.1


		-SA1010:
			`(*regexp.Regexp).FindAll` called with `n == 0`, which will always return zero results
			If n >= 0, the function returns at most n matches/submatches. To
			return all results, specify a negative number.
			Available since
			    2017.1


		-SA1018:
			`strings.Replace` called with `n == 0`, which does nothing
			With n == 0, zero instances will be replaced. To replace all
			instances, use a negative number, or use strings.ReplaceAll.
			Available since
			    2017.1


		-SA1027:
			Atomic access to 64-bit variable must be 64-bit aligned
			On ARM, x86-32, and 32-bit MIPS, it is the caller's responsibility to
			arrange for 64-bit alignment of 64-bit words accessed atomically. The
			first word in a variable or in an allocated struct, array, or slice
			can be relied upon to be 64-bit aligned.
			You can use the structlayout tool to inspect the alignment of fields
			in a struct.
			Available since
			    2019.2


		-SA4018:
			Self-assignment of variables
			Available since
			    2017.1


		-SA4026:
			Go constants cannot express negative zero
			In IEEE 754 floating point math, zero has a sign and can be positive
			or negative. This can be useful in certain numerical code.
			Go constants, however, cannot express negative zero. This means that
			the literals `-0.0` and `0.0` have the same ideal value (zero) and
			will both represent positive zero at runtime.
			To explicitly and reliably create a negative zero, you can use the
			`math.Copysign` function: `math.Copysign(0, -1)`.
			Available since
			    2021.1


		-SA5003:
			Defers in infinite loops will never execute
			Defers are scoped to the surrounding function, not the surrounding
			block. In a function that never returns, i.e. one containing an
			infinite loop, defers will never execute.
			Available since
			    2017.1


		-SA9006:
			Dubious bit shifting of a fixed size integer value
			Bit shifting a value past its size will always clear the value.
			For instance:
			    v := int8(42)
			    v >>= 8
			will always result in 0.
			This check flags bit shifting operations on fixed size integer values only.
			That is, int, uint and uintptr are never flagged to avoid potential false
			positives in somewhat exotic but valid bit twiddling tricks:
			    // Clear any value above 32 bits if integers are more than 32 bits.
			    func f(i int) int {
			        v := i >> 32
			        v = v << 32
			        return i-v
			    }
			Available since
			    2020.2


		-SA1005:
			Invalid first argument to `exec.Command`
			`os/exec` runs programs directly (using variants of the fork and exec
			system calls on Unix systems). This shouldn't be confused with running
			a command in a shell. The shell will allow for features such as input
			redirection, pipes, and general scripting. The shell is also
			responsible for splitting the user's input into a program name and its
			arguments. For example, the equivalent to
			    ls / /tmp
			would be
			    exec.Command("ls", "/", "/tmp")
			If you want to run a command in a shell, consider using something like
			the following – but be aware that not all systems, particularly
			Windows, will have a `/bin/sh` program:
			    exec.Command("/bin/sh", "-c", "ls | grep Awesome")
			Available since
			    2017.1


		-SA1014:
			Non-pointer value passed to `Unmarshal` or `Decode`
			Available since
			    2017.1


		-SA1024:
			A string cutset contains duplicate characters
			The `strings.TrimLeft` and `strings.TrimRight` functions take cutsets, not
			prefixes. A cutset is treated as a set of characters to remove from a
			string. For example,
			    strings.TrimLeft("42133word", "1234"))
			will result in the string `"word"` – any characters that are 1, 2, 3 or
			4 are cut from the left of the string.
			In order to remove one string from another, use `strings.TrimPrefix` instead.
			Available since
			    2017.1


		-SA3001:
			Assigning to `b.N` in benchmarks distorts the results
			The testing package dynamically sets `b.N` to improve the reliability of
			benchmarks and uses it in computations to determine the duration of a
			single operation. Benchmark code must not alter `b.N` as this would
			falsify results.
			Available since
			    2017.1


		-SA1021:
			Using `bytes.Equal` to compare two `net.IP`
			A `net.IP` stores an IPv4 or IPv6 address as a slice of bytes. The
			length of the slice for an IPv4 address, however, can be either 4 or
			16 bytes long, using different ways of representing IPv4 addresses. In
			order to correctly compare two `net.IP`s, the `net.IP.Equal` method should
			be used, as it takes both representations into account.
			Available since
			    2017.1


		-SA4023:
			Impossible comparison of interface value with untyped nil
			Under the covers, interfaces are implemented as two elements, a
			type T and a value V. V is a concrete value such as an int,
			struct or pointer, never an interface itself, and has type T. For
			instance, if we store the int value 3 in an interface, the
			resulting interface value has, schematically, (T=int, V=3). The
			value V is also known as the interface's dynamic value, since a
			given interface variable might hold different values V (and
			corresponding types T) during the execution of the program.
			An interface value is nil only if the V and T are both
			unset, (T=nil, V is not set), In particular, a nil interface will
			always hold a nil type. If we store a nil pointer of type *int
			inside an interface value, the inner type will be *int regardless
			of the value of the pointer: (T=*int, V=nil). Such an interface
			value will therefore be non-nil even when the pointer value V
			inside is nil.
			This situation can be confusing, and arises when a nil value is
			stored inside an interface value such as an error return:
			    func returnsError() error {
			        var p *MyError = nil
			        if bad() {
			            p = ErrBad
			        }
			        return p // Will always return a non-nil error.
			    }
			If all goes well, the function returns a nil p, so the return
			value is an error interface value holding (T=*MyError, V=nil).
			This means that if the caller compares the returned error to nil,
			it will always look as if there was an error even if nothing bad
			happened. To return a proper nil error to the caller, the
			function must return an explicit nil:
			    func returnsError() error {
			        if bad() {
			            return ErrBad
			        }
			        return nil
			    }
			It's a good idea for functions that return errors always to use
			the error type in their signature (as we did above) rather than a
			concrete type such as `*MyError`, to help guarantee the error is
			created correctly. As an example, `os.Open` returns an error even
			though, if not nil, it's always of concrete type *os.PathError.
			Similar situations to those described here can arise whenever
			interfaces are used. Just keep in mind that if any concrete value
			has been stored in the interface, the interface will not be nil.
			For more information, see The Laws of
			Reflection (https://golang.org/doc/articles/laws_of_reflection.html).
			This text has been copied from
			https://golang.org/doc/faq#nil_error, licensed under the Creative
			Commons Attribution 3.0 License.
			Available since
			    2020.2


		-SA4024:
			Checking for impossible return value from a builtin function
			Return values of the `len` and `cap` builtins cannot be negative.
			See https://golang.org/pkg/builtin/#len and https://golang.org/pkg/builtin/#cap.
			Example:
			    if len(slice) < 0 {
			        fmt.Println("unreachable code")
			    }
			Available since
			    2021.1


		-SA5010:
			Impossible type assertion
			Some type assertions can be statically proven to be
			impossible. This is the case when the method sets of both
			arguments of the type assertion conflict with each other, for
			example by containing the same method with different
			signatures.
			The Go compiler already applies this check when asserting from an
			interface value to a concrete type. If the concrete type misses
			methods from the interface, or if function signatures don't match,
			then the type assertion can never succeed.
			This check applies the same logic when asserting from one interface to
			another. If both interface types contain the same method but with
			different signatures, then the type assertion can never succeed,
			either.
			Available since
			    2020.1


		-SA9003:
			Empty body in an if or else branch
			Available since
			    2017.1


		-SA6005:
			Inefficient string comparison with `strings.ToLower` or `strings.ToUpper`
			Converting two strings to the same case and comparing them like so
			    if strings.ToLower(s1) == strings.ToLower(s2) {
			        ...
			    }
			is significantly more expensive than comparing them with
			`strings.EqualFold(s1, s2)`. This is due to memory usage as well as
			computational complexity.
			`strings.ToLower` will have to allocate memory for the new strings, as
			well as convert both strings fully, even if they differ on the very
			first byte. strings.EqualFold, on the other hand, compares the strings
			one character at a time. It doesn't need to create two intermediate
			strings and can return as soon as the first non-matching character has
			been found.
			For a more in-depth explanation of this issue, see
			https://blog.digitalocean.com/how-to-efficiently-compare-strings-in-go/
			Available since
			    2019.2


		-SA3000:
			`TestMain` doesn't call `os.Exit`, hiding test failures
			Test executables (and in turn `go test`) exit with a non-zero status
			code if any tests failed. When specifying your own TestMain function,
			it is your responsibility to arrange for this, by calling `os.Exit` with
			the correct code. The correct code is returned by `(*testing.M).Run`, so
			the usual way of implementing TestMain is to end it with
			`os.Exit(m.Run())`.
			Available since
			    2017.1


		-SA4012:
			Comparing a value against NaN even though no value is equal to NaN
			Available since
			    2017.1


		-SA4017:
			A pure function's return value is discarded, making the call pointless
			Available since
			    2017.1


		-SA5001:
			Deferring `Close` before checking for a possible error
			Available since
			    2017.1


		-SA5008:
			Invalid struct tag
			Available since
			    2019.2


		-SA6001:
			Missing an optimization opportunity when indexing maps by byte slices
			Map keys must be comparable, which precludes the use of byte slices.
			This usually leads to using string keys and converting byte slices to
			strings.
			Normally, a conversion of a byte slice to a string needs to copy the data and
			causes allocations. The compiler, however, recognizes `m[string(b)]` and
			uses the data of `b` directly, without copying it, because it knows that
			the data can't change during the map lookup. This leads to the
			counter-intuitive situation that
			    k := string(b)
			    println(m[k])
			    println(m[k])
			will be less efficient than
			    println(m[string(b)])
			    println(m[string(b)])
			because the first version needs to copy and allocate, while the second
			one does not.
			For some history on this optimization, check out commit
			f5f5a8b6209f84961687d993b93ea0d397f5d5bf in the Go repository.
			Available since
			    2017.1


		-SA6003:
			Converting a string to a slice of runes before ranging over it
			You may want to loop over the runes in a string. Instead of converting
			the string to a slice of runes and looping over that, you can loop
			over the string itself. That is,
			    for _, r := range s {}
			and
			    for _, r := range []rune(s) {}
			will yield the same values. The first version, however, will be faster
			and avoid unnecessary memory allocations.
			Do note that if you are interested in the indices, ranging over a
			string and over a slice of runes will yield different indices. The
			first one yields byte offsets, while the second one yields indices in
			the slice of runes.
			Available since
			    2017.1


		-SA5007:
			Infinite recursive call
			A function that calls itself recursively needs to have an exit
			condition. Otherwise it will recurse forever, until the system runs
			out of memory.
			This issue can be caused by simple bugs such as forgetting to add an
			exit condition. It can also happen "on purpose". Some languages have
			tail call optimization which makes certain infinite recursive calls
			safe to use. Go, however, does not implement TCO, and as such a loop
			should be used instead.
			Available since
			    2017.1


		-SA1006:
			Printf with dynamic first argument and no further arguments
			Using `fmt.Printf` with a dynamic first argument can lead to unexpected
			output. The first argument is a format string, where certain character
			combinations have special meaning. If, for example, a user were to
			enter a string such as
			    Interest rate: 5%
			and you printed it with
			    fmt.Printf(s)
			it would lead to the following output:
			    Interest rate: 5%!(NOVERB).
			Similarly, forming the first parameter via string concatenation with
			user input should be avoided for the same reason. When printing user
			input, either use a variant of fmt.Print, or use the `%s` Printf verb
			and pass the string as an argument.
			Available since
			    2017.1


		-SA1015:
			Using `time.Tick` in a way that will leak. Consider using `time.NewTicker`, and only use `time.Tick` in tests, commands and endless functions
			Available since
			    2017.1


		-SA1016:
			Trapping a signal that cannot be trapped
			Not all signals can be intercepted by a process. Specifically, on
			UNIX-like systems, the `syscall.SIGKILL` and `syscall.SIGSTOP` signals are
			never passed to the process, but instead handled directly by the
			kernel. It is therefore pointless to try and handle these signals.
			Available since
			    2017.1


		-SA1020:
			Using an invalid host:port pair with a `net.Listen`-related function
			Available since
			    2017.1


		-SA4015:
			Calling functions like `math.Ceil` on floats converted from integers doesn't do anything useful
			Available since
			    2017.1


		-SA4016:
			Certain bitwise operations, such as `x ^ 0`, do not do anything useful
			Available since
			    2017.1


		-SA5000:
			Assignment to nil map
			Available since
			    2017.1


		-SA1012:
			A nil `context.Context` is being passed to a function, consider using `context.TODO` instead
			Available since
			    2017.1


		-SA1030:
			Invalid argument in call to a `strconv` function
			This check validates the format, number base and bit size arguments of
			the various parsing and formatting functions in `strconv`.
			Available since
			    2021.1


		-SA4003:
			Comparing unsigned values against negative values is pointless
			Available since
			    2017.1


		-SA4006:
			A value assigned to a variable is never read before being overwritten. Forgotten error check or dead code?
			Available since
			    2017.1


		-SA4021:
			`x = append(y)` is equivalent to `x = y`
			Available since
			    2019.2


		-SA5002:
			The empty for loop (`for {}`) spins and can block the scheduler
			Available since
			    2017.1


		-S1002:
			Omit comparison with boolean constant
			Before:
			    if x == true {}
			After:
			    if x {}
			Available since
			    2017.1


		-S1012:
			Replace `time.Now().Sub(x)` with `time.Since(x)`
			The `time.Since` helper has the same effect as using `time.Now().Sub(x)`
			but is easier to read.
			Before:
			    time.Now().Sub(x)
			After:
			    time.Since(x)
			Available since
			    2017.1


		-S1008:
			Simplify returning boolean expression
			Before:
			    if <expr> {
			        return true
			    }
			    return false
			After:
			    return <expr>
			Available since
			    2017.1


		-S1024:
			Replace `x.Sub(time.Now())` with `time.Until(x)`
			The `time.Until` helper has the same effect as using `x.Sub(time.Now())`
			but is easier to read.
			Before:
			    x.Sub(time.Now())
			After:
			    time.Until(x)
			Available since
			    2017.1
*/
package main

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// “Abstract” syntax representation.

package gc

// A Node is a single node in the syntax tree.
// Actually the syntax tree is a syntax DAG, because there is only one
// node with Op=ONAME for a given instance of a variable x.
// The same is true for Op=OTYPE and Op=OLITERAL.
type Node struct {
	// Tree structure.
	// Generic recursive walks should follow these fields.
	Left  *Node
	Right *Node
	Ninit Nodes
	Nbody Nodes
	List  Nodes
	Rlist Nodes

	// most nodes
	Type *Type
	Orig *Node // original form, for printing, and tracking copies of ONAMEs

	// func
	Func *Func

	// ONAME
	Name *Name

	Sym *Sym        // various
	E   interface{} // Opt or Val, see methods below

	// Various. Usually an offset into a struct. For example, ONAME nodes
	// that refer to local variables use it to identify their stack frame
	// position. ODOT, ODOTPTR, and OINDREG use it to indicate offset
	// relative to their base address. ONAME nodes on the left side of an
	// OKEY within an OSTRUCTLIT use it to store the named field's offset.
	// OXCASE and OXFALL use it to validate the use of fallthrough.
	// Possibly still more uses. If you find any, document them.
	Xoffset int64

	Lineno int32

	// OREGISTER, OINDREG
	Reg int16

	Esc uint16 // EscXXX

	Op          Op
	Nointerface bool
	Ullman      uint8 // sethi/ullman number
	Addable     bool  // addressable
	Etype       EType // op for OASOP, etype for OTYPE, exclam for export, 6g saved reg
	Bounded     bool  // bounds check unnecessary
	Class       Class // PPARAM, PAUTO, PEXTERN, etc
	Embedded    uint8 // ODCLFIELD embedded type
	Colas       bool  // OAS resulting from :=
	Diag        uint8 // already printed error about this
	Noescape    bool  // func arguments do not escape; TODO(rsc): move Noescape to Func struct (see CL 7360)
	Walkdef     uint8
	Typecheck   uint8
	Local       bool
	Dodata      uint8
	Initorder   uint8
	Used        bool
	Isddd       bool // is the argument variadic
	Implicit    bool
	Addrtaken   bool // address taken, even if not moved to heap
	Assigned    bool // is the variable ever assigned to
	Likely      int8 // likeliness of if statement
	Hasbreak    bool // has break statement
	hasVal      int8 // +1 for Val, -1 for Opt, 0 for not yet set
}

// Val returns the Val for the node.
func (n *Node) Val() Val {
	if n.hasVal != +1 {
		return Val{}
	}
	return Val{n.E}
}

// SetVal sets the Val for the node, which must not have been used with SetOpt.
func (n *Node) SetVal(v Val) {
	if n.hasVal == -1 {
		Debug['h'] = 1
		Dump("have Opt", n)
		Fatalf("have Opt")
	}
	n.hasVal = +1
	n.E = v.U
}

// Opt returns the optimizer data for the node.
func (n *Node) Opt() interface{} {
	if n.hasVal != -1 {
		return nil
	}
	return n.E
}

// SetOpt sets the optimizer data for the node, which must not have been used with SetVal.
// SetOpt(nil) is ignored for Vals to simplify call sites that are clearing Opts.
func (n *Node) SetOpt(x interface{}) {
	if x == nil && n.hasVal >= 0 {
		return
	}
	if n.hasVal == +1 {
		Debug['h'] = 1
		Dump("have Val", n)
		Fatalf("have Val")
	}
	n.hasVal = -1
	n.E = x
}

// Name holds Node fields used only by named nodes (ONAME, OPACK, some OLITERAL).
type Name struct {
	Pack      *Node // real package for import . names
	Pkg       *Pkg  // pkg for OPACK nodes
	Heapaddr  *Node // temp holding heap address of param
	Inlvar    *Node // ONAME substitute while inlining
	Defn      *Node // initializing assignment
	Curfn     *Node // function for local variables
	Param     *Param
	Decldepth int32 // declaration loop depth, increased for every loop or label
	Vargen    int32 // unique name for ONAME within a function.  Function outputs are numbered starting at one.
	Iota      int32 // value if this name is iota
	Funcdepth int32
	Method    bool // OCALLMETH name
	Readonly  bool
	Captured  bool // is the variable captured by a closure
	Byval     bool // is the variable captured by value or by reference
	Needzero  bool // if it contains pointers, needs to be zeroed on function entry
	Keepalive bool // mark value live across unknown assembly call
}

type Param struct {
	Ntype *Node

	// ONAME func param with PHEAP
	Outerexpr  *Node // expression copied into closure for variable
	Stackparam *Node // OPARAM node referring to stack copy of param

	// ONAME PPARAM
	Field *Field // TFIELD in arg struct

	// ONAME closure param with PPARAMREF
	Outer   *Node // outer PPARAMREF in nested closure
	Closure *Node // ONAME/PHEAP <-> ONAME/PPARAMREF
}

// Func holds Node fields used only with function-like nodes.
type Func struct {
	Shortname  *Node
	Enter      Nodes // for example, allocate and initialize memory for escaping parameters
	Exit       Nodes
	Cvars      Nodes   // closure params
	Dcl        []*Node // autodcl for this func/closure
	Inldcl     Nodes   // copy of dcl for use in inlining
	Closgen    int
	Outerfunc  *Node
	FieldTrack map[*Sym]struct{}
	Outer      *Node // outer func for closure
	Ntype      *Node // signature
	Top        int   // top context (Ecall, Eproc, etc)
	Closure    *Node // OCLOSURE <-> ODCLFUNC
	FCurfn     *Node
	Nname      *Node

	Inl     Nodes // copy of the body for use in inlining
	InlCost int32
	Depth   int32

	Endlineno int32
	WBLineno  int32 // line number of first write barrier

	Pragma        Pragma // go:xxx function annotations
	Dupok         bool   // duplicate definitions ok
	Wrapper       bool   // is method wrapper
	Needctxt      bool   // function uses context register (has closure variables)
	ReflectMethod bool   // function calls reflect.Type.Method or MethodByName
}

type Op uint8

// Node ops.
const (
	OXXX = Op(iota)

	// names
	ONAME    // var, const or func name
	ONONAME  // unnamed arg or return value: f(int, string) (int, error) { etc }
	OTYPE    // type name
	OPACK    // import
	OLITERAL // literal

	// expressions
	OADD             // Left + Right
	OSUB             // Left - Right
	OOR              // Left | Right
	OXOR             // Left ^ Right
	OADDSTR          // +{List} (string addition, list elements are strings)
	OADDR            // &Left
	OANDAND          // Left && Right
	OAPPEND          // append(List)
	OARRAYBYTESTR    // Type(Left) (Type is string, Left is a []byte)
	OARRAYBYTESTRTMP // Type(Left) (Type is string, Left is a []byte, ephemeral)
	OARRAYRUNESTR    // Type(Left) (Type is string, Left is a []rune)
	OSTRARRAYBYTE    // Type(Left) (Type is []byte, Left is a string)
	OSTRARRAYBYTETMP // Type(Left) (Type is []byte, Left is a string, ephemeral)
	OSTRARRAYRUNE    // Type(Left) (Type is []rune, Left is a string)
	OAS              // Left = Right or (if Colas=true) Left := Right
	OAS2             // List = Rlist (x, y, z = a, b, c)
	OAS2FUNC         // List = Rlist (x, y = f())
	OAS2RECV         // List = Rlist (x, ok = <-c)
	OAS2MAPR         // List = Rlist (x, ok = m["foo"])
	OAS2DOTTYPE      // List = Rlist (x, ok = I.(int))
	OASOP            // Left Etype= Right (x += y)
	OASWB            // Left = Right (with write barrier)
	OCALL            // Left(List) (function call, method call or type conversion)
	OCALLFUNC        // Left(List) (function call f(args))
	OCALLMETH        // Left(List) (direct method call x.Method(args))
	OCALLINTER       // Left(List) (interface method call x.Method(args))
	OCALLPART        // Left.Right (method expression x.Method, not called)
	OCAP             // cap(Left)
	OCLOSE           // close(Left)
	OCLOSURE         // func Type { Body } (func literal)
	OCMPIFACE        // Left Etype Right (interface comparison, x == y or x != y)
	OCMPSTR          // Left Etype Right (string comparison, x == y, x < y, etc)
	OCOMPLIT         // Right{List} (composite literal, not yet lowered to specific form)
	OMAPLIT          // Type{List} (composite literal, Type is map)
	OSTRUCTLIT       // Type{List} (composite literal, Type is struct)
	OARRAYLIT        // Type{List} (composite literal, Type is array or slice)
	OPTRLIT          // &Left (left is composite literal)
	OCONV            // Type(Left) (type conversion)
	OCONVIFACE       // Type(Left) (type conversion, to interface)
	OCONVNOP         // Type(Left) (type conversion, no effect)
	OCOPY            // copy(Left, Right)
	ODCL             // var Left (declares Left of type Left.Type)

	// Used during parsing but don't last.
	ODCLFUNC  // func f() or func (r) f()
	ODCLFIELD // struct field, interface field, or func/method argument/return value.
	ODCLCONST // const pi = 3.14
	ODCLTYPE  // type Int int

	ODELETE    // delete(Left, Right)
	ODOT       // Left.Sym (Left is of struct type)
	ODOTPTR    // Left.Sym (Left is of pointer to struct type)
	ODOTMETH   // Left.Sym (Left is non-interface, Right is method name)
	ODOTINTER  // Left.Sym (Left is interface, Right is method name)
	OXDOT      // Left.Sym (before rewrite to one of the preceding)
	ODOTTYPE   // Left.Right or Left.Type (.Right during parsing, .Type once resolved)
	ODOTTYPE2  // Left.Right or Left.Type (.Right during parsing, .Type once resolved; on rhs of OAS2DOTTYPE)
	OEQ        // Left == Right
	ONE        // Left != Right
	OLT        // Left < Right
	OLE        // Left <= Right
	OGE        // Left >= Right
	OGT        // Left > Right
	OIND       // *Left
	OINDEX     // Left[Right] (index of array or slice)
	OINDEXMAP  // Left[Right] (index of map)
	OKEY       // Left:Right (key:value in struct/array/map literal, or slice index pair)
	OPARAM     // variant of ONAME for on-stack copy of a parameter or return value that escapes.
	OLEN       // len(Left)
	OMAKE      // make(List) (before type checking converts to one of the following)
	OMAKECHAN  // make(Type, Left) (type is chan)
	OMAKEMAP   // make(Type, Left) (type is map)
	OMAKESLICE // make(Type, Left, Right) (type is slice)
	OMUL       // Left * Right
	ODIV       // Left / Right
	OMOD       // Left % Right
	OLSH       // Left << Right
	ORSH       // Left >> Right
	OAND       // Left & Right
	OANDNOT    // Left &^ Right
	ONEW       // new(Left)
	ONOT       // !Left
	OCOM       // ^Left
	OPLUS      // +Left
	OMINUS     // -Left
	OOROR      // Left || Right
	OPANIC     // panic(Left)
	OPRINT     // print(List)
	OPRINTN    // println(List)
	OPAREN     // (Left)
	OSEND      // Left <- Right
	OSLICE     // Left[Right.Left : Right.Right] (Left is untypechecked or slice; Right.Op==OKEY)
	OSLICEARR  // Left[Right.Left : Right.Right] (Left is array)
	OSLICESTR  // Left[Right.Left : Right.Right] (Left is string)
	OSLICE3    // Left[R.Left : R.R.Left : R.R.R] (R=Right; Left is untypedchecked or slice; R.Op and R.R.Op==OKEY)
	OSLICE3ARR // Left[R.Left : R.R.Left : R.R.R] (R=Right; Left is array; R.Op and R.R.Op==OKEY)
	ORECOVER   // recover()
	ORECV      // <-Left
	ORUNESTR   // Type(Left) (Type is string, Left is rune)
	OSELRECV   // Left = <-Right.Left: (appears as .Left of OCASE; Right.Op == ORECV)
	OSELRECV2  // List = <-Right.Left: (apperas as .Left of OCASE; count(List) == 2, Right.Op == ORECV)
	OIOTA      // iota
	OREAL      // real(Left)
	OIMAG      // imag(Left)
	OCOMPLEX   // complex(Left, Right)

	// statements
	OBLOCK    // { List } (block of code)
	OBREAK    // break
	OCASE     // case List: Nbody (select case after processing; List==nil means default)
	OXCASE    // case List: Nbody (select case before processing; List==nil means default)
	OCONTINUE // continue
	ODEFER    // defer Left (Left must be call)
	OEMPTY    // no-op (empty statement)
	OFALL     // fallthrough (after processing)
	OXFALL    // fallthrough (before processing)
	OFOR      // for Ninit; Left; Right { Nbody }
	OGOTO     // goto Left
	OIF       // if Ninit; Left { Nbody } else { Rlist }
	OLABEL    // Left:
	OPROC     // go Left (Left must be call)
	ORANGE    // for List = range Right { Nbody }
	ORETURN   // return List
	OSELECT   // select { List } (List is list of OXCASE or OCASE)
	OSWITCH   // switch Ninit; Left { List } (List is a list of OXCASE or OCASE)
	OTYPESW   // List = Left.(type) (appears as .Left of OSWITCH)

	// types
	OTCHAN   // chan int
	OTMAP    // map[string]int
	OTSTRUCT // struct{}
	OTINTER  // interface{}
	OTFUNC   // func()
	OTARRAY  // []int, [8]int, [N]int or [...]int

	// misc
	ODDD        // func f(args ...int) or f(l...) or var a = [...]int{0, 1, 2}.
	ODDDARG     // func f(args ...int), introduced by escape analysis.
	OINLCALL    // intermediary representation of an inlined call.
	OEFACE      // itable and data words of an empty-interface value.
	OITAB       // itable word of an interface value.
	OSPTR       // base pointer of a slice or string.
	OCLOSUREVAR // variable reference at beginning of closure function
	OCFUNC      // reference to c function pointer (not go func value)
	OCHECKNIL   // emit code to ensure pointer/interface not nil
	OVARKILL    // variable is dead
	OVARLIVE    // variable is alive

	// thearch-specific registers
	OREGISTER // a register, such as AX.
	OINDREG   // offset plus indirect of a register, such as 8(SP).

	// arch-specific opcodes
	OCMP    // compare: ACMP.
	ODEC    // decrement: ADEC.
	OINC    // increment: AINC.
	OEXTEND // extend: ACWD/ACDQ/ACQO.
	OHMUL   // high mul: AMUL/AIMUL for unsigned/signed (OMUL uses AIMUL for both).
	OLROT   // left rotate: AROL.
	ORROTC  // right rotate-carry: ARCR.
	ORETJMP // return to other function
	OPS     // compare parity set (for x86 NaN check)
	OPC     // compare parity clear (for x86 NaN check)
	OSQRT   // sqrt(float64), on systems that have hw support
	OGETG   // runtime.getg() (read g pointer)

	OEND
)

// Nodes is a pointer to a slice of *Node.
// For fields that are not used in most nodes, this is used instead of
// a slice to save space.
type Nodes struct{ slice *[]*Node }

// Slice returns the entries in Nodes as a slice.
// Changes to the slice entries (as in s[i] = n) will be reflected in
// the Nodes.
func (n Nodes) Slice() []*Node {
	if n.slice == nil {
		return nil
	}
	return *n.slice
}

// Len returns the number of entries in Nodes.
func (n Nodes) Len() int {
	if n.slice == nil {
		return 0
	}
	return len(*n.slice)
}

// Index returns the i'th element of Nodes.
// It panics if n does not have at least i+1 elements.
func (n Nodes) Index(i int) *Node {
	return (*n.slice)[i]
}

// First returns the first element of Nodes (same as n.Index(0)).
// It panics if n has no elements.
func (n Nodes) First() *Node {
	return (*n.slice)[0]
}

// Second returns the second element of Nodes (same as n.Index(1)).
// It panics if n has fewer than two elements.
func (n Nodes) Second() *Node {
	return (*n.slice)[1]
}

// Set sets n to a slice.
// This takes ownership of the slice.
func (n *Nodes) Set(s []*Node) {
	if len(s) == 0 {
		n.slice = nil
	} else {
		n.slice = &s
	}
}

// Set1 sets n to a slice containing a single node.
func (n *Nodes) Set1(node *Node) {
	n.slice = &[]*Node{node}
}

// MoveNodes sets n to the contents of n2, then clears n2.
func (n *Nodes) MoveNodes(n2 *Nodes) {
	n.slice = n2.slice
	n2.slice = nil
}

// SetIndex sets the i'th element of Nodes to node.
// It panics if n does not have at least i+1 elements.
func (n Nodes) SetIndex(i int, node *Node) {
	(*n.slice)[i] = node
}

// Addr returns the address of the i'th element of Nodes.
// It panics if n does not have at least i+1 elements.
func (n Nodes) Addr(i int) **Node {
	return &(*n.slice)[i]
}

// Append appends entries to Nodes.
// If a slice is passed in, this will take ownership of it.
func (n *Nodes) Append(a ...*Node) {
	if n.slice == nil {
		if len(a) > 0 {
			n.slice = &a
		}
	} else {
		*n.slice = append(*n.slice, a...)
	}
}

// AppendNodes appends the contents of *n2 to n, then clears n2.
func (n *Nodes) AppendNodes(n2 *Nodes) {
	switch {
	case n2.slice == nil:
	case n.slice == nil:
		n.slice = n2.slice
	default:
		*n.slice = append(*n.slice, *n2.slice...)
	}
	n2.slice = nil
}

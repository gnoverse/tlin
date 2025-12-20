package minilogic

// Expr represents an expression in the MiniLogic system.
type Expr interface {
	isExpr()
	String() string
}

// LiteralExpr represents a literal value (int, bool, string, nil).
type LiteralExpr struct {
	Val Value
}

func (LiteralExpr) isExpr() {}
func (e LiteralExpr) String() string {
	return e.Val.String()
}

// VarExpr represents a variable reference.
type VarExpr struct {
	Name string
}

func (VarExpr) isExpr() {}
func (e VarExpr) String() string {
	return e.Name
}

// BinaryOp represents binary operators.
type BinaryOp int

const (
	_ BinaryOp = iota
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpEq
	OpNeq
	OpLt
	OpLte
	OpGt
	OpGte
	OpAnd
	OpOr
)

func (op BinaryOp) String() string {
	switch op {
	case OpAdd:
		return "+"
	case OpSub:
		return "-"
	case OpMul:
		return "*"
	case OpDiv:
		return "/"
	case OpMod:
		return "%"
	case OpEq:
		return "=="
	case OpNeq:
		return "!="
	case OpLt:
		return "<"
	case OpLte:
		return "<="
	case OpGt:
		return ">"
	case OpGte:
		return ">="
	case OpAnd:
		return "&&"
	case OpOr:
		return "||"
	default:
		return "?"
	}
}

// BinaryExpr represents a binary expression.
type BinaryExpr struct {
	Op    BinaryOp
	Left  Expr
	Right Expr
}

func (BinaryExpr) isExpr() {}
func (e BinaryExpr) String() string {
	return "(" + e.Left.String() + " " + e.Op.String() + " " + e.Right.String() + ")"
}

// UnaryOp represents unary operators.
type UnaryOp int

const (
	OpNot UnaryOp = iota
	OpNeg
)

func (op UnaryOp) String() string {
	switch op {
	case OpNot:
		return "!"
	case OpNeg:
		return "-"
	default:
		return "?"
	}
}

// UnaryExpr represents a unary expression.
type UnaryExpr struct {
	Op      UnaryOp
	Operand Expr
}

func (UnaryExpr) isExpr() {}
func (e UnaryExpr) String() string {
	return "(" + e.Op.String() + e.Operand.String() + ")"
}

// CallExpr represents a function call expression.
// Function calls are treated as opaque and may have side effects.
type CallExpr struct {
	Func string
	Args []Expr
}

func (CallExpr) isExpr() {}
func (e CallExpr) String() string {
	result := e.Func + "("
	for i, arg := range e.Args {
		if i > 0 {
			result += ", "
		}
		result += arg.String()
	}
	return result + ")"
}

// Stmt represents a statement in the MiniLogic system.
type Stmt interface {
	isStmt()
	String() string
}

// AssignStmt represents an assignment statement: x = e
type AssignStmt struct {
	Var  string
	Expr Expr
}

func (AssignStmt) isStmt() {}
func (s AssignStmt) String() string {
	return s.Var + " = " + s.Expr.String()
}

// DeclAssignStmt represents a short variable declaration: x := e
// Used primarily in if-init statements.
type DeclAssignStmt struct {
	Var  string
	Expr Expr
}

func (DeclAssignStmt) isStmt() {}
func (s DeclAssignStmt) String() string {
	return s.Var + " := " + s.Expr.String()
}

// SeqStmt represents a sequence of statements: S1 ; S2
type SeqStmt struct {
	First  Stmt
	Second Stmt
}

func (SeqStmt) isStmt() {}
func (s SeqStmt) String() string {
	return s.First.String() + "; " + s.Second.String()
}

// IfStmt represents a conditional statement: if [init]; cond { then } else { els }
type IfStmt struct {
	Init Stmt // optional initializer (can be nil)
	Cond Expr
	Then Stmt
	Else Stmt // can be nil for if without else
}

func (IfStmt) isStmt() {}
func (s IfStmt) String() string {
	result := "if "
	if s.Init != nil {
		result += s.Init.String() + "; "
	}
	result += s.Cond.String() + " { " + s.Then.String() + " }"
	if s.Else != nil {
		result += " else { " + s.Else.String() + " }"
	}
	return result
}

// ReturnStmt represents a return statement: return e?
type ReturnStmt struct {
	Value Expr // can be nil for bare return
}

func (ReturnStmt) isStmt() {}
func (s ReturnStmt) String() string {
	if s.Value == nil {
		return "return"
	}
	return "return " + s.Value.String()
}

// BreakStmt represents a break statement.
type BreakStmt struct{}

func (BreakStmt) isStmt() {}
func (BreakStmt) String() string {
	return "break"
}

// ContinueStmt represents a continue statement.
type ContinueStmt struct{}

func (ContinueStmt) isStmt() {}
func (ContinueStmt) String() string {
	return "continue"
}

// CallStmt represents a function call statement: call(...)
// Used for opaque side-effecting calls.
type CallStmt struct {
	Call *CallExpr
}

func (CallStmt) isStmt() {}
func (s CallStmt) String() string {
	return s.Call.String()
}

// BlockStmt represents a block of statements.
type BlockStmt struct {
	Stmts []Stmt
}

func (BlockStmt) isStmt() {}
func (s BlockStmt) String() string {
	if len(s.Stmts) == 0 {
		return "{}"
	}
	result := "{ "
	for i, stmt := range s.Stmts {
		if i > 0 {
			result += "; "
		}
		result += stmt.String()
	}
	return result + " }"
}

// NoopStmt represents an empty/no-op statement.
type NoopStmt struct{}

func (NoopStmt) isStmt() {}
func (NoopStmt) String() string {
	return "noop"
}

// Helper functions to construct AST nodes

// Lit creates a literal expression from a value.
func Lit(v Value) Expr {
	return LiteralExpr{Val: v}
}

// IntLit creates an integer literal expression.
func IntLit(v int64) Expr {
	return LiteralExpr{Val: IntValue{Val: v}}
}

// BoolLit creates a boolean literal expression.
func BoolLit(v bool) Expr {
	return LiteralExpr{Val: BoolValue{Val: v}}
}

// StrLit creates a string literal expression.
func StrLit(v string) Expr {
	return LiteralExpr{Val: StringValue{Val: v}}
}

// NilLit creates a nil literal expression.
func NilLit() Expr {
	return LiteralExpr{Val: NilValue{}}
}

// Var creates a variable reference expression.
func Var(name string) Expr {
	return VarExpr{Name: name}
}

// Assign creates an assignment statement.
func Assign(v string, e Expr) Stmt {
	return AssignStmt{Var: v, Expr: e}
}

// DeclAssign creates a short variable declaration.
func DeclAssign(v string, e Expr) Stmt {
	return DeclAssignStmt{Var: v, Expr: e}
}

// Seq creates a sequence of statements.
func Seq(stmts ...Stmt) Stmt {
	if len(stmts) == 0 {
		return NoopStmt{}
	}
	if len(stmts) == 1 {
		return stmts[0]
	}
	result := stmts[len(stmts)-1]
	for i := len(stmts) - 2; i >= 0; i-- {
		result = SeqStmt{First: stmts[i], Second: result}
	}
	return result
}

// If creates an if statement without init.
func If(cond Expr, then, els Stmt) Stmt {
	return IfStmt{Cond: cond, Then: then, Else: els}
}

// IfInit creates an if statement with init.
func IfInit(init Stmt, cond Expr, then, els Stmt) Stmt {
	return IfStmt{Init: init, Cond: cond, Then: then, Else: els}
}

// Return creates a return statement.
func Return(e Expr) Stmt {
	return ReturnStmt{Value: e}
}

// ReturnVoid creates a bare return statement.
func ReturnVoid() Stmt {
	return ReturnStmt{}
}

// Break creates a break statement.
func Break() Stmt {
	return BreakStmt{}
}

// Continue creates a continue statement.
func ContinueS() Stmt {
	return ContinueStmt{}
}

// Call creates a call statement.
func Call(fn string, args ...Expr) Stmt {
	return CallStmt{Call: &CallExpr{Func: fn, Args: args}}
}

// Block creates a block statement.
func Block(stmts ...Stmt) Stmt {
	return BlockStmt{Stmts: stmts}
}

// Binary creates a binary expression.
func Binary(op BinaryOp, left, right Expr) Expr {
	return BinaryExpr{Op: op, Left: left, Right: right}
}

// Unary creates a unary expression.
func Unary(op UnaryOp, operand Expr) Expr {
	return UnaryExpr{Op: op, Operand: operand}
}

// Not creates a logical not expression.
func Not(e Expr) Expr {
	return UnaryExpr{Op: OpNot, Operand: e}
}

// And creates a logical and expression.
func And(left, right Expr) Expr {
	return BinaryExpr{Op: OpAnd, Left: left, Right: right}
}

// Or creates a logical or expression.
func Or(left, right Expr) Expr {
	return BinaryExpr{Op: OpOr, Left: left, Right: right}
}

// Eq creates an equality expression.
func Eq(left, right Expr) Expr {
	return BinaryExpr{Op: OpEq, Left: left, Right: right}
}

// Neq creates a not-equal expression.
func Neq(left, right Expr) Expr {
	return BinaryExpr{Op: OpNeq, Left: left, Right: right}
}

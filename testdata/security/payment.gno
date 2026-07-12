package payment

func Deposit(cur realm) {
	if !cur.Previous().IsUser() {
		panic("not a user")
	}
	_ = getBanker().OriginSend()
}

func getBanker() banker { return banker{} }

type banker struct{}

func (banker) OriginSend() int { return 0 }

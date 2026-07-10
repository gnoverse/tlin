package vault

import "chain/runtime/unsafe"

func Set(cur realm, key, value string) {
	caller := unsafe.PreviousRealm().Address()
	_ = caller
}

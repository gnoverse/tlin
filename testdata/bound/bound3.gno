package main

func removeStringFromStringArr(arr []string, str string) []string {
	for i, a := range arr {
		if a == str {
			return append(arr[:i], arr[i+1:]...)
		}
	}
	return arr
}

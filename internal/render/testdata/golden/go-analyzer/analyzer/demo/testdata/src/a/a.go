package a

func Empty(cond bool) {
	if cond { // want "empty branch body"
	}
}

func NotEmpty(cond bool) {
	if cond {
		println("fine")
	}
}

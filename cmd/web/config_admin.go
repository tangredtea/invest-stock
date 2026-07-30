package main

func validAdminUsername(u string) bool {
	return len(u) >= minAdminUserLen && len(u) <= maxAdminUserLen
}

func validAdminPassword(p string) bool {
	return len(p) >= minAdminPassLen && len(p) <= maxAdminPassLen
}

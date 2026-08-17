//go:build desktop && windows

package app

/*
void appWinRoleCut(void);
void appWinRoleCopy(void);
void appWinRolePaste(void);
void appWinRoleSelectAll(void);
*/
import "C"

func winRole(r Role) {
	switch r {
	case RoleCut:
		C.appWinRoleCut()
	case RoleCopy:
		C.appWinRoleCopy()
	case RolePaste:
		C.appWinRolePaste()
	case RoleSelectAll:
		C.appWinRoleSelectAll()
	}
}

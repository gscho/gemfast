package middleware

import (
	"fmt"
	"os"
	u "github.com/gemfast/server/internal/utils"
	"github.com/casbin/casbin/v2"
	"github.com/rs/zerolog/log"
)

var	ACL casbin.Enforcer


func init() {
	var policyPath string
	var authPath string
	
	for _, path := range []string{"/opt/gemfast/etc/gemfast/gemfast_acl.csv", "gemfast_acl.csv"} {
		exists, _ := u.FileExists(path)
		if exists {
			policyPath = path
			break
		}
	}

	for _, path := range []string{"/opt/gemfast/etc/gemfast/auth_model.conf", "auth_model.conf"} {
		exists, _ := u.FileExists(path)
		if exists {
			authPath = path
			break
		}
	}
	if policyPath == "" || authPath == "" {
		log.Error().Err(fmt.Errorf("unable to locate auth_model and gemfast_acl")).Msg("failed to find acl files")
		os.Exit(1)
	}
	acl, err := casbin.NewEnforcer(authPath, policyPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to initialize the acl")
		os.Exit(1)
	}
	ACL = *acl
}
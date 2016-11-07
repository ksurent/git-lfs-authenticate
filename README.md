# git-lfs-authenticate

SSH authentication shim for git-lfs.

There are several reasons you might want to use this despite the fact that the
spec recommends against using SSH:

* You want to auth against a different host than your LFS server
* You want to have password-less authentication
    * Re-using existing SSH keys infrastructure
    * Not wanting to store unencrypted passwords in memory (credentials cache)
* You want to have authorisation based on LDAP group membership
    * Re-using existing RBAC infrastructure

## Installation

Build and put somewhere inside PATH (e.g. /usr/local/bin/).

## Configuration

By default git-lfs-authenticate reads its configuration data from
/etc/git-lfs-authenticate.conf. You can override this by providing the
GIT_LFS_AUTHENTICATE_CONFIG environment variable.

This repository contains a sample config file in example.conf. Check it out.

Below is a short explanation of every option:

* Lfs.Url — address of LFS server in URL format
* Lfs.User — shared username that will be used to talk to LFS on behalf of the user
* Lfs.Password — password for the shared user

* Ldap.Urls — comma–separated list of LDAP servers in URL format
* Ldap.Groups — comma–separated list of LDAP groups
* Ldap.Cacert — path to cacert.pem (for when using TLS)
* Ldap.Base — base DN for LDAP searches

LDAP servers are tried one–by–one in randomised order until either a match is
found or a fatal error occurs (e.g. user not found).

The list of LDAP groups is OR-ed, i.e. a membership in at least one of the
groups is sufficient.

The cacert file is optional. It’s still possible to use TLS without it but
without hostname verification.

## Usage

This command will be invoked via the git-lfs client. See
[the spec](https://github.com/github/git-lfs/tree/master/docs/api) for more
details.

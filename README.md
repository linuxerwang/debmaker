# debmaker
A debian package maker programmed in Golang.

It has the following features which might meet your special needs:
- describe package build with a spec.
- the spec can be read from a file or standard input.
- specify control attributes in the spec file.
- specify src folder files/dirs and their position in the deb file (the actually install place).
- auto generate the md5sum file.
- auto calculate the disk size of the installed package.
- all package content belong to root, no need to use fakeroot.

To get the source code and build the debmake binary:

$ go get github.com/linuxerwang/debmaker

To run the example:

$ debmaker -spec-file example/deb.spec -version 1.0.5 -arch amd64

#!/bin/env python

from subprocess import call

def main():
	call(["/bin/bash", "/files/restore_dep.sh"])
	call(["/usr/local/go/bin/go", "test", "github.com/kopwei/goovs"])
	

if __name__ == "__main__":
	main()
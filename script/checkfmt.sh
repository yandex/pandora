#!/bin/bash


test_fmt() {
    DIR="$1"
    hash goimports 2>&- || { echo >&2 "goimports not in PATH."; exit 1; }

    for file in $(find -L $DIR -type f -name "*.go" -not -path "./Godeps/*")
    do
        output=`cat $file | goimports -l 2>&1`
        if test $? -ne 0
        then
            output=`echo "$output" | sed "s,<standard input>,$file,"`
            syntaxerrors="${list}${output}\n"
        elif test -n "$output"
        then
            list="${list}${file}\n"
        fi
    done
    exitcode=0
    if test -n "$syntaxerrors"
    then
        echo >&2 "goimports found syntax errors:"
        printf "$syntaxerrors"
        exitcode=1
    fi
    if test -n "$list"
    then
        echo >&2 "goimports needs to format these files (run make fmt and git add):"
        printf "$list"
        printf "\n"
        exitcode=1
    fi
    exit $exitcode
}

main() {
    test_fmt "$@"
}

main "$@"

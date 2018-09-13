#!/bin/bash -e
SED=sed

# # first rename all directories
# for i in $(find . -type d | grep -v "^./.git" | grep gochain); do 
#     git mv $i $(echo $i| $SED "s/gochain/indigo/")
#     git mv $i $(echo $i| $SED "s/Gochain/Indigo/")
# done

# # then rename all files
# for i in $(find . -type f | grep -v "^./.git" | grep gochain); do
#     git mv $i $(echo $i| $SED "s/gochain/indigo/")
#     git mv $i $(echo $i| $SED "s/Gochain/Indigo/")
# done

# now replace all gochain references to the new coin name
for i in $(find . -type f | grep -v "^./.git"); do
    # $SED -i "s/Gochain/Indigo/g" $i
    # $SED -i "s/GoChain/Indigo/g" $i
    # $SED -i "s/gochain/indigo/g" $i
    # $SED -i "s/GOCHAIN/INDIGO/g" $i
    $SED -i "s/github.com\/indigo-io\//github.com\/fulcrumchain\//g" $i
done
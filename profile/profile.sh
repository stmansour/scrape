#!/bin/bash
# This script processes the html pages from step1 into a list of 
# URLs that can be called to get detail info on each person.

declare -a link_filters=(
	's/^.{80}//'
	's/^ *<[^>]+>//'
	's/<.*LoadPerson\)/(LoadPerson)/'
	's/".*//'
	's/\(LoadPerson/; \(LoadPerson/'
	)

rm -f step4.csv

for x in {a..z}
do
	for y in {a..z}
	do
		fbase="${x}${y}"
		if [ -f step1/${fbase}.html ]; then
			echo "Phase 2:  ${fbase}"
			cat step1/${fbase}.html | tail -n +36 |sed -n -e :a -e '1,3!{P;N;D;};N;ba'| sed -e '/<[\/]*table>/d'| sed -e '/page-break-before:always/d' > x
			for f in "${link_filters[@]}"
			do
				perl -pe "$f" x > x1; mv x1 x
			done
			cat x |grep "(LoadPerson)" >> step4.csv
		fi
	done
done

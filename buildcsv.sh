#!/bin/bash

#  Anomolies:
# "Fitzgerald, William" was not found in faa.csv but was in faa.scrape
# several lines in faa.scrape still had <COL:x>.  
# Remove all TESTERS - a few of these were left in the data
#
#  The base site with this info is: https://directory.faa.gov/appsPub/National/employeedirectory/faadir.nsf

#----------------------------------------------------
#  First, grab all the html files from the FAA site...
#----------------------------------------------------
rm -rf html
mkdir html
pushd html
../form/form
popd

#----------------------------------------------------
#  Next, process every html file... convert to CSV
#----------------------------------------------------
for x in {a..z}
do
	for y in {a..z}
	do
		echo "Phase 1: $x$y"
		python html2csv.py html/${x}${y}.html
	done
done

#----------------------------------------------------
#  Filter out the cruft...
#----------------------------------------------------
cat html/*.csv | grep -v "close window" | grep -v \"Vacant\" | egrep -v "^\" \"$" | egrep -v '^"[^"]*","[^"]*","[^"]*","[^"]*"$' | egrep -v '^"[^"]*","[^"]*"$' | egrep -v '^$' | egrep -v '"Travel,' | egrep -v '"Test,'  > faa.csv

#-----------------------------------------------------------
#  Rescan for the links to get each person's profile url
#-----------------------------------------------------------
declare -a link_filters=(
	# this one actually isolates the link we want
	's/<COL:1>([^<]+)<[^\[]+\[\[([^\]]+)\].*/"$1",$2/'
	# this one discards the opening link that appears on some lines.
	's/^\[\[[^\]]+\]\[([^\]]+)\]\]/$1/'	
	's/^"\[\[[^\]]+\]\[([^\]]+)\]\]"/"$1"/'
# paranoid, apply again
	's/^\[\[[^\]]+\]\[([^\]]+)\]\]/$1/'	
	's/^"\[\[[^\]]+\]\[([^\]]+)\]\]"/"$1"/'
	)

rm -f faa.scrape

for x in {a..z}
do
	for y in {a..z}
	do
		fbase="${x}${y}"
		echo "Phase 2:  ${fbase}"
		cat html/${fbase}.html | ../scrape | grep "LoadPerson" > y
		for f in "${link_filters[@]}"
		do
			perl -pe "$f" y > y1; mv y1 y
		done

		cat y >> faa.scrape
	done
done

#------------------------------------------------------------------------
# not sure why some of the lines got missed, but we need to reapply the 
# filter to remove the opening link. Some of the lines didn't take...
#------------------------------------------------------------------------
perl -pe 's/^"\[\[[^\]]+\]\[([^\]]+)\]\]"/"$1"/' faa.scrape >y
mv y faa.scrape

#------------------------------------------------------------------------
# now we need to follow the links and grab each person's profile...
#------------------------------------------------------------------------
pushd csvbld
rm -f final.csv
./csvbld ../faa.scrape

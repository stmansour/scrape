#!/bin/bash
STARTTIME=$(date)
MYSQL=$(sh -c "which mysql")
MYSQLDUMP=$(sh -c "which mysqldump")
QUICK=0

usage() {
    cat <<ZZEOF
Usage: $0 options...
Optons:
    -q           quick mode. Go only once through every loop.

Description:     Generate a new FAA directory in the database and create 
                 a csv file that can be imported into Excel.

ZZEOF
    exit 1
}

while getopts ":q" o; do
    case "${o}" in
        q)
            QUICK=1
            ;;
        *)
            usage
            ;;
    esac
done

if [ ${QUICK} -gt 0 ]; then
	QUICKOPT="-q"
fi

#----------------------------------------------------
#  Start with a clean workspace
#----------------------------------------------------
rm -rf workspace
mkdir -p workspace/step1
mkdir -p workspace/step2

#----------------------------------------------------
#  Step 1 - Grab the basic list from the faa site
#           Using a single thread, this step takes about 40 min
#----------------------------------------------------
pushd workspace/step1
../../bin/form -w 50 ${QUICKOPT}
STEP1=$(date)

#--------------------------------------------------------
#  Step 2 - process every step1 file... convert to CSV
#--------------------------------------------------------
for x in {a..z}
do
	for y in {a..z}
	do
		echo "Phase 1: $x$y"
		python ../../bin/html2csv.py ${x}${y}.html
		if [ ${QUICK} -gt 0 ]; then break; fi
	done
	if [ ${QUICK} -gt 0 ]; then break; fi
done
mv *.csv ../step2/
STEP2=$(date)

#-----------------------------------------------------------------
#  Step 3 - Filter out cruft and aggregate to a single csv file...
#  When this step completes, step3.csv will have the basic directory
#  information. This will be used to create the database. 
#  There are still further details that we can get
#  from the Profile on each person. We'll get this info in step 4.
#-----------------------------------------------------------------
cd ../step2
cat *.csv | grep -v "close window" | grep -v \"Vacant\" | egrep -v "^\" \"$" | egrep -v '^"[^"]*","[^"]*","[^"]*","[^"]*"$' | egrep -v '^"[^"]*","[^"]*"$' | egrep -v '^$' | egrep -v '"Travel,' | egrep -v '"Test,'  > ../step3.csv
cd ../
${MYSQL} --no-defaults < ../bin/schema.sql
../bin/loadnames -f step3.csv
STEP3=$(date)

#-----------------------------------------------------------------
#  Step 4 - Filter the raw html files to find the profile link for
#           each person.  Call the profile link, pull down the html
#           parse out the useful data and update the db 
#-----------------------------------------------------------------
../bin/profile.sh ${QUICKOPT}
STEP4=$(date)

#-----------------------------------------------------------------
#  Step 5 - Process every profile link
#-----------------------------------------------------------------
../bin/csvbld -b ../bin -w 50 ${QUICKOPT}
STEP5=$(date)

#-----------------------------------------------------------------
#  Step 6 - Export to csv
#-----------------------------------------------------------------
cat >xxyyzz <<EOF
USE faa
describe people;
EOF
${MYSQL} --no-defaults <xxyyzz >x
rm -f xxyyzz
cat x | sed 1,2d | awk '{printf "%s,", $1} END {printf "\n"}' | sed 's/,$//' > head.csv

rm -rf tmp
mkdir tmp
chmod 777 tmp

${MYSQLDUMP} --no-defaults -t -T./tmp faa --fields-enclosed-by=\" --fields-terminated-by=,
cat head.csv ./tmp/people.txt >faadir.csv

STEP6=$(date)

echo "Completed"
echo "Started............: ${STARTTIME}"
echo "Step 1 completed...: ${STEP1}"
echo "Step 2 completed...: ${STEP2}"
echo "Step 3 completed...: ${STEP3}"
echo "Step 4 completed...: ${STEP4}"
echo "Step 5 completed...: ${STEP5}"
echo "Step 6 completed...: ${STEP6}"


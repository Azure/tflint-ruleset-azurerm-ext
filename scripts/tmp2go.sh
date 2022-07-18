files=$(ls $1)
for filename in $files
do
  newfilename=${filename%.tmp}
  mv $1/${filename} $1/${newfilename}
done

#!/usr/bin/env bash

diffFiles=$(git diff --name-only HEAD^ HEAD)
error=false

# shellcheck disable=SC2068
for doc in ${diffFiles[@]};
do
  dirname=$(dirname "$doc")
  category=$(basename "$dirname")
  case "$category" in
    "d" | "r")
      grep "https://doc.ksyun.com/)\.$" "$doc" > /dev/null
      if [[ "$?" == "0" ]]; then
        echo -e "\033[31mDoc :${doc}: Please input the exact link. Currently it is https://doc.ksyun.com/. \033[0m"
        error=true
      fi
      ;;
    "alicloud")
      grep "fmt.Println" "$doc" > /dev/null
      if [[ "$?" == "0" ]]; then
        echo -e "\033[31mFile :${doc}: Please Remove the fmt.Println Method! \033[0m"
        error=true
      fi
    ;;
  esac
done

if $error; then
  exit 1
fi

exit 0
dpkg-query -W -f='${Status}|${Package}|${Architecture}|${Version}|${Provides}\n' 2>/dev/null | grep '^install ok installed|'

There are two ways of specifying the separate debuginfo file:
	1) The executable contains a debug link that specifies the name of the separate debuginfo file.
	   The separate debug file’s name is usually executable.debug, where executable is the name of the corresponding executable file without leading directories (e.g., ls.debug for /usr/bin/ls).
	2) The executable contains a build ID, a unique bit string that is also present in the corresponding debuginfo file.
	   (This is supported only on some operating systems, when using the ELF or PE file formats for binary files and the GNU Binutils.)
	   The debuginfo file’s name is not specified explicitly by the build ID, but can be computed from the build ID, see below.

    So, for example, suppose you ask Agent to debug /usr/bin/ls, which has a debug link that specifies the file ls.debug,
	and a build ID whose value in hex is abcdef1234.
	If the list of the global debug directories includes /usr/lib/debug (which is the default),
	then Finder will look for the following debug information files, in the indicated order:

		- /usr/lib/debug/.build-id/ab/cdef1234.debug
		- /usr/bin/ls.debug
		- /usr/bin/.debug/ls.debug
		- /usr/lib/debug/usr/bin/ls.debug

For further information, see: https://sourceware.org/gdb/onlinedocs/gdb/Separate-Debug-Files.html

A debug link is a special section of the executable file named .gnu_debuglink. The section must contain:
A filename, with any leading directory components removed, followed by a zero byte,
 - zero to three bytes of padding, as needed to reach the next four-byte boundary within the section, and
 - a four-byte CRC checksum, stored in the same endianness used for the executable file itself.
The checksum is computed on the debugging information file’s full contents by the function given below,
passing zero as the crc argument.

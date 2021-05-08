# escape=`

ARG BASE_TAG=latest_1803

FROM mback2k/windows-buildbot-tools:${BASE_TAG}

ARG MINGW_GET_PACKAGE=mingw-get-0.6.2-mingw32-beta-20131004-1-bin.zip
ADD ${MINGW_GET_PACKAGE} C:\Windows\Temp\${MINGW_GET_PACKAGE}

RUN Start-Process -FilePath "C:\Program` Files\7-Zip\7z.exe" -ArgumentList x, "C:\Windows\Temp\${MINGW_GET_PACKAGE}", `-oC:\MinGW -NoNewWindow -PassThru -Wait; `
    Remove-Item @('C:\Windows\Temp\*', 'C:\Users\*\Appdata\Local\Temp\*') -Force -Recurse;

# We are using forward slashes, because mingw-get is also using them.
RUN C:/MinGW/bin/mingw-get.exe update; `
    C:/MinGW/bin/mingw-get.exe install msys-base msys-zip gcc g++ libtool cmake; `
    C:/MinGW/bin/mingw-get.exe install mingw32-make mingw32-libz mingw32-zlib;



#
#FROM mmozeiko/mingw-w64
#
#ADD . /app
#
#WORKDIR /app
#
#RUN  git submodule init && git submodule update
#
#RUN cd libmdbx
#
#RUN export
#RUN cd libmdbx && cmake -DCMAKE_SYSTEM_NAME=Windows \
#                        -DCMAKE_INSTALL_PREFIX=${MINGW} \
#                        -DCMAKE_FIND_ROOT_PATH=${MINGW} \
#                        -DCMAKE_FIND_ROOT_PATH_MODE_PROGRAM=NEVER \
#                        -DCMAKE_FIND_ROOT_PATH_MODE_LIBRARY=ONLY \
#                        -DCMAKE_FIND_ROOT_PATH_MODE_INCLUDE=ONLY \
#                        -DCMAKE_C_COMPILER=x86_64-w64-mingw32-gcc \
#                        -DCMAKE_CXX_COMPILER=x86_64-w64-mingw32-g++ \
#                        -DCMAKE_RC_COMPILER=x86_64-w64-mingw32-windres .
#
#RUN cd libmdbx && cmake --build .
#



#FROM dockcross/windows-static-x64-posix
#
#ADD . /app
#
#WORKDIR /app
#
#RUN  git submodule init && git submodule update
#
#RUN cd libmdbx
#
#RUN export
#RUN cd libmdbx && cmake -DCMAKE_CXX_COMPILER_TARGET=Windows -DCMAKE_C_COMPILER=x86_64-w64-mingw32-gcc -DCMAKE_SYSTEM_NAME=Windows -DCMAKE_C_COMPILER_WORKS=1 -DCMAKE_C_FLAGS:STRING="-Wno-unused-variable -Wno-unused-parameter" .
#RUN cd libmdbx && cmake --build .







#FROM x1unix/go-mingw:1.16
#
#RUN apk add git cmake bash sed
#
#ADD . /app
#
#WORKDIR /app
#
#RUN  git submodule init && git submodule update
#
#RUN cd libmdbx
#
#ENV CXX_FOR_TARGET=x86_64-w64-mingw32-g++
#ENV CC_FOR_TARGET=x86_64-w64-mingw32-gcc
#ENV CC=x86_64-w64-mingw32-gcc
#ENV CMAKE_C_COMPILER=x86_64-w64-mingw32-gcc
#ENV CMAKE_CXX_COMPILER_TARGET=Windows
#ENV CMAKE_SYSTEM_NAME=Windows
#ENV CMAKE_C_COMPILER_WORKS=1
#
#RUN export
#RUN cd libmdbx && cmake -DCMAKE_CXX_COMPILER_TARGET=Windows -DCMAKE_C_COMPILER=x86_64-w64-mingw32-gcc -DCMAKE_SYSTEM_NAME=Windows -DCMAKE_C_COMPILER_WORKS=1 -DCMAKE_C_FLAGS:STRING="-Wno-unused-variable -Wno-unused-parameter" .
#RUN cd libmdbx && cmake --build .




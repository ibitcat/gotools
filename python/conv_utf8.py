#!/usr/bin/env python
# -*- coding: utf-8 -*-

import os, sys;
import getopt, chardet;

# 需要安装chardet ：pip install chardet
# 改良自：https://github.com/owent-utils/python/blob/master/format/conv_to_utf8.py


""" run as a executable """

enable_bom = False

def getDirList(p):
    p = str( p )
    if p=="":
        return [ ]

    p = p.replace( "/","\\")
    if p[ -1] != "\\":
        p = p+"\\"

    a = os.listdir( p )
    b = [ p+x for x in a ]
    return b

def convFile(file_path):
    stats_conv_num = 0
    stats_fix_bom_num = 0
    stats_skip_num = 0
    stats_failed_num = 0

    try: 
        f = open(file_path, 'rb')
    except Exception:
        stats_failed_num = stats_failed_num + 1
        print('open file ' + file_path + ' failed')
        return
    
    file_buffers = f.read()
    f.close()
    detected_info = chardet.detect(file_buffers)
    if detected_info['encoding'] is not None:
        if detected_info['encoding'].lower() == 'utf-8':
            if enable_bom and file_buffers[0:3] == b"\xEF\xBB\xBF":
                stats_skip_num = stats_skip_num + 1
                print(file_path + ' is already utf-8 with bom, skiped')
            elif not enable_bom and file_buffers[0:3] != b"\xEF\xBB\xBF":
                stats_skip_num = stats_skip_num + 1
                print(file_path + ' is already utf-8 without bom, skiped.')
            elif enable_bom:
                f = open(file_path, 'wb')
                f.write(file_buffers.decode(detected_info['encoding']).encode("UTF-8-SIG"))
                f.close()
                print(file_path + ' add utf-8 bom done')
                stats_fix_bom_num = stats_fix_bom_num + 1
            else:
                f = open(file_path, 'wb')
                f.write(file_buffers[3:])
                f.truncate()
                f.close()
                print(file_path + ' remove utf-8 bom done')
                stats_fix_bom_num = stats_fix_bom_num + 1
        else:
            try: 
                if enable_bom and file_buffers[0:3] == b"\xEF\xBB\xBF":
                    print(file_path + ' is already utf-8 with bom, skiped')
                    return
                elif not enable_bom and file_buffers[0:3] == b"\xEF\xBB\xBF":
                    new_buffers = file_buffers.decode(detected_info['encoding']).encode('utf-8')
                else:
                    new_buffers = file_buffers.decode(detected_info['encoding']).encode('UTF-8-SIG')

                f = open(file_path, 'wb')
                suffix = 'witout bom'
                if enable_bom:
                    suffix = 'with bom'
                
                f.write(new_buffers)
                f.close()
                
                print('OK! ' + detected_info['encoding'] + ' -> utf-8 ' + suffix + ' success. File= ' + file_path)
                stats_conv_num = stats_conv_num + 1
            except Exception:
                stats_failed_num = stats_failed_num + 1
                print('Fail! ' + detected_info['encoding'] + ' -> utf-8 ' + suffix + ' failed. File= ' + file_path)
    else:
        print(file_path + ' can not detect encoding')
        stats_skip_num = stats_skip_num + 1

    #print('all jobs done. convert number: {0}, fix bom number: {1}, skip number: {2}, failed number: {3}'.format(file_path,stats_conv_num, stats_fix_bom_num, stats_skip_num, stats_failed_num))

def stepPath(left_args):
    if len(left_args) > 0:
        for file_path in left_args:
            if os.path.isfile(file_path):
                sufix = os.path.splitext(file_path)[1]
                if sufix==".h" or sufix==".cpp":
                    convFile(file_path)
            elif os.path.isdir(file_path):
                child = getDirList(file_path)
                stepPath(child)

if __name__ == "__main__":
    def print_help_msg():
        print('usage: ' + sys.argv[0] + ' [options...] [file paths...]')
        print('options:')
        print('-h, --help                               help messages')
        print('-v, --version                            show version and exit')
        print('-b, --with-bom                           utf-8 with bom')


    opts, left_args = getopt.getopt(sys.argv[1:], 'bhv', [
        'help',
        'version',
        'with-bom'
    ])

    for opt_key, opt_val in opts:
        if opt_key in ('-b', '--with-bom'):
            enable_bom = True
        elif opt_key in ('-h', '--help'):
            print_help_msg()
            exit(0)
        elif opt_key in ('-v', '--version'):
            print('1.0.0.0')
            exit(0)

    stepPath(left_args)

    print('')

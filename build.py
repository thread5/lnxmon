import os
import shutil
import sys
import traceback


# BUILD_DIR = './build'
# if os.path.exists(BUILD_DIR):
#     shutil.rmtree(BUILD_DIR)
# os.mkdir(BUILD_DIR)


highcharts_new = ''
f = None
try:
    f = open('./static/highcharts.js')
    highcharts_new = f.read()
except:
    print(traceback.format_exc())
    sys.exit(-1)
finally:
    if f is not None:
        f.close()
highcharts_new = '<script>\n%s</script>' % (highcharts_new)


highcharts_old = '<script type="text/javascript" src="./static/highcharts.js"></script>'
f, f2 = None, None
try:
    f = open('./templates/index.html')
    f2 = open('./build/index.html', 'w')
    lines = f.readlines()
    for line in lines:
        line = line.rstrip()
        if line.endswith(highcharts_old):
            f2.write(highcharts_new)
        else:
            f2.write(line)
        f2.write(os.linesep)
except:
    print(traceback.format_exc())
    sys.exit(-1)
finally:
    if f is not None:
        f.close()
    if f2 is not None:
        f2.close()


html_new = ''
f = None
try:
    f = open('./build/index.html')
    html_new = f.read()
except:
    print(traceback.format_exc())
    sys.exit(-1)
finally:
    if f is not None:
        f.close()
html_new = 'HTML := `%s`' % (html_new)


html_old = 'HTML := ""'
f, f2 = None, None
try:
    f = open('./lnxmonsrv.go')
    f2 = open('./build/lnxmonsrv.go', 'w')
    lines = f.readlines()
    for line in lines:
        line = line.rstrip()
        if line.endswith(html_old):
            f2.write(html_new)
        else:
            f2.write(line)
        f2.write(os.linesep)
except:
    print(traceback.format_exc())
    sys.exit(-1)
finally:
    if f is not None:
        f.close()
    if f2 is not None:
        f2.close()


f, f2 = None, None
try:
    f = open('./lnxmoncli.go')
    f2 = open('./build/lnxmoncli.go', 'w')
    lines = f.readlines()
    for line in lines:
        line = line.rstrip()
        f2.write(line)
        f2.write(os.linesep)
except:
    print(traceback.format_exc())
    sys.exit(-1)
finally:
    if f is not None:
        f.close()
    if f2 is not None:
        f2.close()


cmd = 'rm -f ./build/index.html'
exit_status = os.system(cmd)
if exit_status != 0:
    print('Exit status is %d' % exit_status)

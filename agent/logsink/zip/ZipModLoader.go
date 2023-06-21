package zip

import (
//"github.com/whatap/go-api/agent/agent/config"
)

type ZipModLoader struct {
	zipImpl ZipMod
	libpath string
}

func NewZipModLoader() *ZipModLoader {
	p := new(ZipModLoader)
	p.zipImpl = NewDefaultZipMod()
	return p

}

//func (this ZipModLoader) Load(){
//	ConfLogSink := config.GetConfig().ConfLogSink
//
//	if !ConfLogSink.LogSinkZipEnabled {
//		return
//	}
//
//	thisPath := ConfLogSink.LogSinkZipLibpath
//
//
//		String thispath = ConfLogSink.logsink_zip_libpath;
//		if (StringUtil.isEmpty(thispath)) {
//			if (zipImpl == null || libpath != null) {
//				libpath = null;
//				zipImpl = new DefaultZipMod();
//			}
//			return;
//		}
//		if (thispath.equals(libpath))
//			return;
//
//		try {
//			File file = new File(thispath);
//			if (file.exists() == false) {
//				Logger.println("LogSink ZipModule load fail: " + file + " is not exist");
//				return;
//			}
//			String main = getMain(file);
//			if (StringUtil.isEmpty(main)) {
//				Logger.println("LogSink ZipModule load fail: no mainClass is defined");
//				return;
//			}
//			URL[] urls = new URL[] { file.toURI().toURL() };
//			URLClassLoader loader = new URLClassLoader(urls, ZipModLoader.class.getClassLoader());
//			ZipMod z = (ZipMod) Class.forName(main, true, loader).newInstance();
//			zipImpl = z;
//			libpath = thispath;
//			Logger.println("LogSink ZipModule load success: " + thispath + " " + main);
//		} catch (Throwable t) {
//			Logger.println("LogSink ZipModule load fail: " + t);
//		}
//		if (zipImpl == null) {
//			zipImpl = new DefaultZipMod();
//		}
//	}

package trace

import (
	//"log"
	"github.com/whatap/golib/util/hmap"
	"github.com/whatap/go-api/agent/util/logutil"
	"strings"
)

// agnet.boot -> agent.trace로 이동 import cycle error
type SearchPathMap struct {
	targetAnnotation []string
	//pathParamUrlSet       *hmap.StringLinkedSet
	noPathParamUrlHashSet *hmap.IntLinkedSet
}

// TODO 임시로 java class 생성 사용
// ServiceURLPatternDetector 에서 Anntation 관련 검색을 진행할 경우
// 테스트 후 사용 필요
type ClassLoader struct {
}

type Class struct {
}

func (this *Class) getClassLoader() *ClassLoader {
	return nil
}
func (this *Class) getName() string {
	return ""
}
func (this *Class) getDeclaredAnnotations() []Annotation {
	return nil
}

type Annotation struct {
}

func NewSearchPathMap() *SearchPathMap {
	p := new(SearchPathMap)
	p.targetAnnotation = make([]string, 10)

	// TODO StringLinkedSet
	//p.pathParamUrlSet = hmap.NewStringLinkedSet().SetMax(1000)
	p.noPathParamUrlHashSet = hmap.NewIntLinkedSet().SetMax(10000)

	p.targetAnnotation = append(p.targetAnnotation, "org.springframework.stereotype.Controller")
	p.targetAnnotation = append(p.targetAnnotation, "org.springframework.web.bind.annotation.RestController")

	return p
}

func ScanSearchPathMap(clazz *Class) {

	if clazz.getClassLoader() == nil {
		return
	}

	if strings.HasPrefix(clazz.getName(), "whatap") {
		return
	}

	if strings.HasPrefix(clazz.getName(), "org.springframework") {
		return
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				logutil.Println("WA10300", " Recover", r)
			}
		}()

		an := clazz.getDeclaredAnnotations()
		if an == nil {
			return
		}

		//		for (int j = 0; j < len(an); j++) {
		//			if (targetAnnotation.contains(an[j].annotationType().getName())) {
		//				process(clazz, an);
		//				break;
		//			}
		//		}
	}()
}

func processSearchPathMap(class1 *Class, an []Annotation) {
	//	classPath := ""
	//	for j := 0; j < an.length; j++) {
	//		if (an[j].annotationType().getName().endsWith(".RequestMapping")) {
	//			String[] s = getRequestMapValue(an[j]);
	//			if (ArrayUtil.isEmpty(s) == false) {
	//				classPath = s[0];
	//			}
	//		}
	//	}
	//	boolean isParamedClassPath = classPath.indexOf('{') >= 0;
	//	java.lang.reflect.Method[] m = class1.getDeclaredMethods();
	//	for (int i = 0; i < m.length; i++) {
	//		Annotation[] methodAnno = m[i].getDeclaredAnnotations();
	//		if (methodAnno == null || methodAnno.length == 0)
	//			continue;
	//		for (int j = 0; j < methodAnno.length; j++) {
	//			if (methodAnno[j].annotationType().getName().endsWith(".RequestMapping")) {
	//				String[] methodPath = getRequestMapValue(methodAnno[j]);
	//				for (int mi = 0; mi < methodPath.length; mi++) {
	//					String path = classPath + methodPath[mi];
	//
	//					if (isParamedClassPath || methodPath[mi].indexOf('{') >= 0) {
	//						pathParamUrlSet.put(path);
	//					} else {
	//						noPathParamUrlHashSet.put(path.hashCode());
	//					}
	//				}
	//				break;
	//			}
	//		}
	//	}
}

func getRequestMapValue(an *Annotation) []string {
	var ret []string

	defer func() {
		if r := recover(); r != nil {
			logutil.Println("WA10301", " Recover", r)
			ret = make([]string, 0)
		}
	}()

	//		valMethod := an.annotationType().getMethod("value", new Class[0]);
	//		ret = []string(valMethod.invoke(an, new Object[0]));
	//

	return ret
}

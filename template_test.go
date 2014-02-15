package revel

import (
  "fmt"
  "testing"
)

func ExampleTemplateAndEnvironment() {
  instance := new(templateAndEnvironment)
  instance.engineName = "anyEngineName"
  instance.templateSet = nil
  instance.methods = map[string]interface{}{"firstMethodName": "firstLambda", "secondMethodName": "secondLambda"}

  fmt.Println(*instance)
  //Output: {anyEngineName <nil> map[firstMethodName:firstLambda secondMethodName:secondLambda]}
}

func TestSetupTemplateEngine(t *testing.T) {
  templatesAndEngine := new(templateAndEnvironment)
  templatesAndEngine.setupTemplateEngine()
  if nameInConfig, ok := Config.String("template.engine"); ok && templatesAndEngine.engineName != nameInConfig {
    t.Error("Name is not set.")
  } else if !ok && templatesAndEngine.engineName != defaultTemplateEngineName {
    t.Error("Name is not set to default.")
  }

  if templatesAndEngine.methods == nil {
    t.Error("Methods are not set.")
  }
  for _, methodName := range [...]string{"initialAddAndParse", "addAndParse", "lookup"} {
    if _, ok := templatesAndEngine.methods[methodName]; !ok {
      t.Error("Method " + methodName + " is not set.")
    }
  }
}

func TestSetTemplateEngineName_blank(t *testing.T) {
  templatesAndEngine := new(templateAndEnvironment)
  templatesAndEngine.setTemplateEngineName("")
  if templatesAndEngine.engineName != defaultTemplateEngineName {
    t.Error("Name is not set.")
  }
}

func TestSetTemplateEngineName(t *testing.T) {
  templatesAndEngine := new(templateAndEnvironment)
  templatesAndEngine.setTemplateEngineName("anySetTemplateEngineName")
  if templatesAndEngine.engineName != "anySetTemplateEngineName" {
    t.Error("Name is not set.")
  }
}

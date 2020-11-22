local json = require("json")

local T={}

T["Query.todoTitles"]=function(parents, args)
  local res=graphql([[query Q1($f:Int!){queryTodo (first:$f){
    title
    id
  }}]],{a=1,b="two",c=false,f=3})
  local t={}
  print("PPP",res)
  for k,v in res.queryTodo() do
    table.insert(t, "Title: "..v.title.." of "..v.id)
  end
  return json.encode(t)
end

T["Query.todoTitles2"]=function(parents, args)
  local t={}
  local res=dql([[{
  todos(func:type(Todo)) {
    id: uid
    title: Todo.title
  }
}]],{a=1,b="two",c=false,f=2})
  for k,v in pairs(json.decode(res).todos) do
      table.insert(t, "Title: "..v.title.." of "..v.id)
  end
  return json.encode(t)
end

T["Mutation.newTodoTitles"]=function(parents, args)
  return json.encode({"title: "..args.title})  
end


T["User.fullName"]=function(parents, args)
  local t={}
  for k,v in parents() do
    table.insert(t,v.firstName.." "..v.lastName)
  end
  return json.encode(t)
end

function onRequest(t)
  local f=T[t.resolver]
  if f~=nil then
    return f(t.parents, t.args)
  end
  return "Not found\n"
end


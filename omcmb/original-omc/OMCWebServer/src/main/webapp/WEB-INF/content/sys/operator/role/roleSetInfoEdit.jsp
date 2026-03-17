<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
	.headContent{
		font-size:14px;
		color:#363B4E;
		font-weight:700;
		display:flex;
		align-items:center;
	}
	.headContent img{
		margin-right:14px;
	}
</style>
<div class="slidebarTitleDiv">
	<span class="slideTitle"><%=rb.getString("JueSe")%></span>
	<div class="slideIcon el-icon el-icon-close" onclick='cancelOperModifyRoleDiv()'></div>
</div>
<div class="slideBody">
	<div class="slideCont">
		<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
			<div class="headContent">
				<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("JiBenXinXi") %>
			</div>
		</div>
		<div class="deviceLogContainer inputformat">
			<p>
				<label><%=rb.getString("JueSeMingCheng")%><%=rb.getString("MaoHao")%></label>
				<input id='roleNameInput' readonly  type='text' style='padding-left:10px;width:390px;'/>
			</p>
			<p style='margin-top:20px'>
				<label><%=rb.getString("MiaoShu")%><%=rb.getString("MaoHao")%></label>
				<textarea class="border-box border"  id='roleDesc' style="resize:none;margin-top:5px;width:390px;height:125px;padding-left:10px;padding-top:5px;"></textarea>
			</p>
		</div>
		<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
			<div class="headContent">
				<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("GongNengQuanXianLieBiao") %>
			</div>
		</div>
		<div class="deviceLogContainer" style='margin-top:15px;'>
			<p id='selectP' class='errorTip' style='color:#CC0000;visibility:hidden'><%=rb.getString("QingXuanZeGongNengQuanXian")%></p>
			<div id="op_new_feature" style="width: 100%;">
				<el-ctree ref='featureTree' style="height: 500px;"
					title="<%=rb.getString("QuanXianLieBiao")%>"
					:default-expanded-keys="['0']"
					:data="data1"
					:cascade="['wForms > forms']"
					:check-forms="[
						{key: 'forms',label: '<%=rb.getString("ZhiDuQuanXuan")%>',prop: 'checked'},
						{key: 'wForms',label: '<%=rb.getString("KeXieQuanXuan")%>',prop: 'write'}
					]"
					:ignore="{
						forms: ['1','7','10','30','32','38','64','66','76','81','92','96','97','98'],
						wForms: ['1','58','59','60','63']
					}"></el-ctree>
			</div>
		</div>
		<div class="onlyLocalShow">
			<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
				<div class="headContent">
					<img src="${ctx}/css/images/global/settingBetter.png"><%=rb.getString("YunYingShangLieBiao") %>
				</div>
			</div>
			<div class="deviceLogContainer" style='margin-top:8px;margin-bottom:20px;'>
				<div class="queryGroup">
					<input id="operCodeName"  name="operator_code" style="width:400px;" placeholder="<%=rb.getString("YunYingShangMingCheng")%>" />
					<b class='el-icon el-icon-common-search' onclick="$('#operModifyRoleOperator').datagrid('reload')"></b>
				</div>
				<div style='height:424px;border:1px solid #E9E9E9;margin-top:20px;'>
					<table  id='operModifyRoleOperator'></table>
				</div>
			</div>
		</div>
	</div>
	<%-- <div class='success'><%=rb.getString("XiuGaiJueSeChengGong")%></div> --%>
</div>
<div class="slideFooter">			
	<span class="el-button el-button--primary" onclick="operSaveModifyRoleSet()" ><%=rb.getString("QueDing")%></span>
	<span class="el-button" onclick="cancelOperModifyRoleDiv()"><%=rb.getString("QuXiao")%></span>
</div>
<script>
	var opvm = new Vue({
		el: '#op_new_feature',
		data() {
			return {
				data1: []
			}
		},
		methods: {
			init(data) {
				var vm = this;
				
				vm.data1 = data;
	        	vm.$nextTick(function(){
	        		vm.$refs.featureTree.reviewForms(vm.data1);
	        	});
			},
			getResult() {
				var vm = this;
					res = vm.$refs.featureTree.getResult(),
					menu_ids = [{id: '1',write: true}];
				
				res.wForms.map(function(id){
					id!='0' && menu_ids.push({id: id,write: true});
				});
				res.forms.map(function(id){
					if(!res.wForms.includes(id) && id!='0'){
						menu_ids.push({id: id,write: false})
					}
				})
				
				return menu_ids;
			}
		}
	});
	var role = $('#operRoleSetListTable').datagrid("getSelections")[0];
	var roleId = $('#operRoleSetListTable').datagrid("getSelections")[0].role_id;
	var codeArr = [];
	var cloneArr = [];
	$(function(){
		closeLoading();
		
		if(isCloudCore == "true" && roleId == "2"){
			$(".onlyLocalShow").hide();
		}else{
			$(".onlyLocalShow").show();
		}
		
		var params = {
		  	"operator_code":$('#operCodeName').val(),
			"role_id":role.role_id,
			"type":"modify",
			 "isAll":1
		 }
		$.post("${ctx}/system/superRoleSet/getOperatorListByRole.action",params,function(data){
  	  		//dataArr = data.rows;
  	  		data.rows.map(function(item,index){
  	  			if(item["check"] == "true"){
  	  				codeArr.push(item["operator_code"])
  	  			}
  	  		})
  	  	    cloneArr = $.extend(true,[],codeArr);
	  	  	$("#operModifyRoleOperator").datagrid({
	 			url:'${ctx}/system/superRoleSet/getOperatorListByRole.action',
	 			queryParams : {
	 				"operator_code":$('#operCodeName').val(),
	 				"role_id":role.role_id,
	 				"type":"modify"
	 				},
	 			singleSelect:false,
	 			fit:true,
	 			fitColumns:true,
	 			border:false,
	 			rownumbers:true,
	 			idField:"operator_code",
	 			pagePosition:'bottom',
	 			pagination: true,
	 			striped: true,
	 			onLoadSuccess:loadSuccessModify,
	 			checkOnSelect:role.role_id == 1?false:true,
	 			onBeforeLoad:beforeload_operModify,
	 			onCheck:checkOperatorFn,
	 			onCheckAll:checkOperatorFnAll,
	 			onUncheck:uncheckOperatorFn,
	 			onUncheckAll:uncheckOperatorFnAll,
	 			columns: [[
	 				{field: 'ck', checkbox: true}, 
	 				{field: 'operator_code',hidden:"true"},
	 				{field: 'operator_name',width:100,title:'<%=rb.getString("YunYingShangMingCheng")%>'},
	 				{field: 'cloud_key',width:100,title:'CloudKey'},
	 			]]
	 		});
  	  	},"json")
		$('#roleNameInput').val(role.role_name);
		$('#roleDesc').val(role.role_desc); 
		$.post("${ctx}/system/superRoleSet/getFeature.action",{
			"role_id": role.role_id,
			"type":"modify"
		},function(data){
        	opvm.init(data);
  	  	},"json")
  	    
  	//键盘回车事件  --- 
  	$("#operCodeName").bind("keyup", function(e){
  		if (e.keyCode == 13){
  			$('#operModifyRoleOperator').datagrid('reload');
  		}
  	}); 
	})
	function proccessData(data){
		data.map(function(item){
    		if(item.children && item.children.length>0) {
    			proccessData(item.children);
    		}else{
    			delete item.state;
    		}
    	});
    	return data;
	}
	function loadSuccessModify(data){
		if(role.role_id == 1){
			$(".onlyLocalShow input[type='checkbox']").map(function(index,item){
				item.disabled = true;
			})
			data.rows.map(function(item,index){
				if(codeArr.indexOf(item["operator_code"])!=-1){
					$("#operModifyRoleOperator").datagrid('checkRow',index);
				}
			})
		}else{
			data.rows.map(function(item,index){
				if(cloneArr.indexOf(item["operator_code"])!= -1){
					$("#operModifyRoleOperator").datagrid('checkRow',index);
				}
			})
		}
	}
	function checkOperatorFn(index,row){
		  if(role.role_id!=1){
			 if(cloneArr.indexOf(row["operator_code"]) == -1){
				 cloneArr.push(row["operator_code"]);
			 }
		  } 
	}
	function uncheckOperatorFn(index,row){
		if(role.role_id!=1){
			if(cloneArr.indexOf(row["operator_code"])!=-1){
				cloneArr.splice(cloneArr.indexOf(row["operator_code"]),1);
			}
			if(cloneArr.length == 0){
				$('#selectP').css('visibility','hidden'); 
			}
		}
	}
	function checkOperatorFnAll(rows){
		if(role.role_id!=1){
			for (var i = 0; i < rows.length; i++) {
				checkOperatorFn("", rows[i]);
		    }
		}
	}
	function uncheckOperatorFnAll(rows){
		if(role.role_id!=1){
			for (var i = 0; i < rows.length; i++) {
				uncheckOperatorFn("", rows[i]);
		    }
		}
	}
	
	function operSaveModifyRoleSet(){
		 var menuArr = [];
		 menuArr = opvm.getResult();
		 if(role.role_id == 1){
			 if(menuArr.length == 0){
				 $('#selectP').css('visibility','visible');
				 $('#operModifyRoleDiv').animate({
						scrollTop:200
					},500);
				 return;
			 } 
		 }else{
			 if(cloneArr.length != 0){
				 if(menuArr.length == 0){
					 $('#selectP').css('visibility','visible');
					 $('#operModifyRoleDiv').animate({
							scrollTop:200
						},500);
					 return;
				 }else{
					 $('#selectP').css('visibility','hidden'); 
				 }
			 }else{
				 $('#selectP').css('visibility','hidden');  
			 }
		 }
		  var checkStr = [];
		  if(role.role_id != 1){
			  checkStr = cloneArr;
		  }else{
			  checkStr = codeArr;
		  }
		  
		 var params = {
				  "role_id":role.role_id,
				  "role_desc":$("#roleDesc").val(),
				  "menu_ids":menuArr,
				  "operator_codes":checkStr
				  
			 }
		  params = JSON.stringify(params);
		  $.post("${ctx}/system/superRoleSet/save.action",{params:params},function(data){
			   if(data["success"]){
				  $("#operRoleSetListTable").datagrid("reload");
				  showMsg('success_msg',data["message"]);
						var  time = setTimeout(function(){
							cancelOperModifyRoleDiv();
					})
			  }else{
				  showMsg('error_msg',data["message"]);
			  }
		  },'json')
	}
	function beforeload_operModify(param){
		param["operator_code"] = $('#operCodeName').val();
	}
</script>
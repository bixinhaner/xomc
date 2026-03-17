<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
	.errorMesRole {
		visibility:hidden;
		color:#CC0000;
	}
	#syseNBSourceDiv .tree-node {
	  background: #fff !important;
	  cursor:pointer;
	  padding:5px 0px 5px 0px;
	}
	.success{
		background:#E9FBFF url("${ctx}/images/success.png") no-repeat 16px center;
		height:38px;
		float:left;
		margin-left:20px;
		margin-top:30px;
		color:#508D9B;
		font-size:15px;
		font-weight:bold;
		padding:0 20px 0 20px;
		line-height:38px;
		text-indent:25px;
		display:none;
	}
</style>
<div style='margin-left:60px;'>
	<div class="omcPageTitleDiv_contain">
			<ul class="omcPageTitleContainer_contain" style="width:80%">
				<li class="active"><%=rb.getString("JiBenXinXi")%></li>
			</ul>
	</div>
	<div class="deviceLogContainer inputformat">
		<p>
			<label><%=rb.getString("JueSeMingCheng")%><%=rb.getString("MaoHao")%></label>
			<input style='padding-left:10px;width:390px' id='sysAddRoleNameInput' onblur="checkSysRoleName()"  type='text'/>
			<p id='sysAddRoleName' class='errorMesRole'><%=rb.getString("QingShuRuJueSeMingCheng")%></p>
		</p>
		<p style='margin-top:5px'>
			<label><%=rb.getString("MiaoShu")%><%=rb.getString("MaoHao")%></label>
			<textarea id="sysAddRoleTextare" style="resize:none;padding-top:5px;padding-left:10px;width:390px;height:125px;margin-top:5px;"></textarea>
		</p>
	</div>
	<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
		<ul class="omcPageTitleContainer_contain" style="width:80%">
			<li class="active"><%=rb.getString("GongNengQuanXianLieBiao")%></li>
		</ul>
	</div>
	<div class="deviceLogContainer" style='margin-top:10px;'>
		<p id='sysRoleSelectMes' class='errorMesRole'><%=rb.getString("QingXuanZeGongNengQuanXian")%></p>
		<div id='sysFeatureLimitDiv' class="tree-lines" style='user-select:none;height:450px;width:787px;border:1px solid #CCE1EF;margin-top:10px;overflow:auto'>
			<table style='height:100%;' id='sysAddFeatureLimit'></table>
		</div>
	</div>
	<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
		<ul class="omcPageTitleContainer_contain" style="width:80%">
			<li class="active"><%=rb.getString("ZiYuanQuanXianLieBiao")%></li>
		</ul>
	</div>
	<div class="deviceLogContainer" style='margin-top:10px;'>
		<p id='sysAddSourceMes' class='errorMesRole'><%=rb.getString("QingXuanZeZiYuanQuanXian")%></p>
		<div class="queryGroup">
			<input  id='addDeviceName' style="width:450px;" placeholder="<%=rb.getString("Title_SheBeiMingCheng")%>" />
			<b class="el-icon el-icon-common-search" onclick="filterSysAddRole()"></b>
		</div>
		<div id='syseNBSourceDiv' class="tree-lines" style='padding-left:10px;padding-top:10px;user-select:none;height:440px;width:777px;border:1px solid #CCE1EF;margin-top:10px;overflow:auto'>
			<ul id='sysAddeNBSourceUl'></ul>
		</div>
	</div>
	<div class="windowButtonGroup" style="float:left !important;margin:30px 0 20px 58px">			
		<a id='saveButtonRole' class="linkbutton linkbutton_trend" onclick="sysSaveAddRole()"><span><%=rb.getString("QueDing")%></span></a>
		<a class="linkbutton linkbutton_nowanna" onclick="cancelSysAddRole();"><span><%=rb.getString("QuXiao")%></span></a>
	</div>
	<div class='success'><%=rb.getString("XinJianJueSeChengGong")%></div>
</div>
<script>
	if($("#curr_operator_span").text() == ""){
		var operator_code = "default";
	}else{
		var operator_code =operator_code;
	}
	var isRolePass = true;
	$(function(){
  	    $('#sysAddFeatureLimit').treegrid({
	        url:"${ctx}/sys/role/getFeature.action",
	        queryParams:{
				"type":"add",
				"operator_code":operator_code
	        },
	        idField: 'id',            //定义关键字段来标识树节点。也就是数据的id
	        treeField: 'text',        //定义树形显示字段
	        fitColumns:true,
	        animate:true,
	        columns: [[                //定义表格头名称
	            {
	                title: '<%=rb.getString("QuanXianLieBiao")%>',
	                field: 'text',
	                width:50,
	                formatter:sysAddTreeMenuFormat
	            },
	            {
	                title: '<%=rb.getString("GongNengCaoZuo")%>',
	                field: 'operation',
	                width: 50,
	                formatter: sysAddOnlyReadAndWriteFormat
	            }
	        ]],
	        onLoadSuccess:function(node,data){
	        	$("#sysFeatureLimitDiv table .datagrid-btable:first>tbody>tr>td>div>.tree-file").removeClass("tree-file").addClass("tree-folder");
	        	var treetable = $(this).treegrid('getPanel').find('div.datagrid-body').parent().parent().parent();
	        	treetable.css('border-width','0px');
	        	var tr = $(this).treegrid('getPanel').find('div.datagrid-body tr.datagrid-row');
	        	 tr.each(function(){
	        		var td = $(this).children('td');
	        		td.css({
	        			'border-width':'0px'
	        		})
	        	});
	        	 var rootId = $('#sysAddFeatureLimit').treegrid('getRoot').id;
	        	 review(rootId);
	        },
	        onBeforeSelect:function(){
	        	return false;
	        },
	        onSelect:function(){
	        	return false;
	        },
	        loadFilter: proccessData,
	        onClickRow:function(row){
	        	$('#sysAddFeatureLimit').treegrid("toggle", row.id);
	        } ,
	        onExpand:function(row){
	      		var root = $('#sysAddFeatureLimit').treegrid('getRoot');
	      		if(row.id!=root.id){
	      			var children = $('#sysAddFeatureLimit').treegrid('getRoot').children;
	      			children.map(function(item,index){
	      				if(item.id == row.id){
	      					$('#sysAddFeatureLimit').treegrid("expand", item.id);
	      				}else{
	      					$('#sysAddFeatureLimit').treegrid("collapse", item.id);
	      				}
	      			})
	      		}
	      	}
	     });
  	  $('#sysAddeNBSourceUl').tree({
			url:"${ctx}/sys/role/getDeviceGroup.action",
			queryParams:{
				"type":"add",
				"operator_code":operator_code
	        },
			checkbox:true,
			idField: 'id',            //定义关键字段来标识树节点。也就是数据的id
			treeField: 'text',        //定义树形显示字段
			fitColumns:true,
			animate:true,
			loadFilter: proccessData,
			onLoadSuccess:function(node,data){
				var root = $('#sysAddeNBSourceUl').tree("getRoots");
				var children = $('#sysAddeNBSourceUl').tree("getChildren",root);
				children.map(function(item,index){
					var isLeaf = $('#sysAddeNBSourceUl').tree("isLeaf",item.target);
					if(!isLeaf && index == 1 || index == 0){
						$('#sysAddeNBSourceUl').tree("expand",item.target);
					}else{
						$('#sysAddeNBSourceUl').tree("collapse",item.target);
					}
				})
			} ,
			 onBeforeSelect:function(){
		        	return false;
		        },
		     onSelect:function(){
		        	return false;
		        },
			onClick:function(node){
				$('#sysAddeNBSourceUl').tree("toggle", node.target);
			},
			onCheck:function(){
				$("#sysAddSourceMes").css("visibility","hidden");
			},
			onExpand:function(node){
				var nodeAll = $('#sysAddeNBSourceUl').tree("getRoots")[0].children;
				nodeAll.map(function(item,index){
					if(node.id == item.id){
							$('#sysAddeNBSourceUl').tree("expand",node.target);
					}else{
						$('#sysAddeNBSourceUl').tree("collapse",item.target);
					}
				})
			}
			});
  	  $("#addDeviceName").bind("keyup",function(e){
  		  if(e.keyCode == 13){
  			filterSysAddRole();
  		  }
  	  })
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
	function sysAddOnlyReadAndWriteFormat(value,row,rowIndex){
		var rowId = row.id,
			pId = row.pid,
			rowText = row.text,
			exclude = "Dashboard,首页";
		if(row.children && row.children.length>0){
			//return "";
			return ["<span readid = "+rowId+" readpid = "+pId+" onclick = 'accessTree(\"sysRoleSelectMes\",event,this,\""+rowText+"\",\""+exclude+"\")'  class='tree-checkbox tree-checkbox0 readOper'></span>",
				    "<span class='tree-title'><%=rb.getString("ZhiDuQuanXuan")%></span>",
				    "<span writeid = "+rowId+" writepid = "+pId+" style='cursor:pointer;margin-left:5px;' onclick='sysRoleCheckboxToggle(event,this,"+rowId+")' class='tree-checkbox tree-checkbox0 writeOper'></span>",
				    "<span class='tree-title'><%=rb.getString("KeXieQuanXuan")%></span>"].join("");
		}else{
			if(row.text == "首页"||row.text == "Dashboard"){
				return "<span style='display:none;cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox1 readOper '></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span style='cursor:pointer;margin-left:5px;display:none' class='tree-checkbox tree-checkbox1 writeOper'></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiXie")%></span>";
			}else{
				if(row.text == "EPC"||row.text == "告警确认"||row.text == "Alarm Confirm"||row.text == "告警清除"||row.text == "Clear Alarm"||row.text == "告警恢复"||row.text == "Restore Alarm"||row.text == "告警删除"||row.text == "Delete Alarm"||row.text == "用户向导"||row.text == "User Guide"||row.text == "关于"||row.text == "About"
					||row.text == "同步"||row.text == "Synchronize"||row.text == "关联的CPEs"||row.text == "CPEs"
						||row.text == "基站IMSI信息清除"||row.text == "Clear IMSI"||row.text == "系统资源监控"||row.text == "Resource"||row.text == "设置"||row.text == "Setting"||row.text == "激活"||row.text == "Active"||row.text == "eGW"||row.text == "网关"||row.text == "UPS"||row.text == "APN"||row.text == "快速配置"||row.text == "Quick Configuration"){
					return "<span  style='display:none;cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox0 readOper '></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span writeid="+rowId+" writepid="+pId+"  style='cursor:pointer;margin-left:5px;display:none' onclick='sysRoleCheckboxToggle(event,this,"+rowId+")' class='tree-checkbox tree-checkbox0 writeOper'></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiXie")%></span>"	
				}else{
					if(row.reWrite == "true"){
						return "<span style='cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox0 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span writeid="+rowId+" writepid='"+ pId +"' style='cursor:pointer;margin-left:5px;' onclick='sysRoleCheckboxToggle(event,this,"+rowId+")' class='tree-checkbox tree-checkbox0 writeOper'></span><span class='tree-title'><%=rb.getString("ZhiXie")%></span>"
					}else{
						return "<span style='cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox0 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%>";
					}
				}
			}
		}
	}
	function sysAddTreeMenuFormat(value,row,rowIndex){
		var rowText = row.text;
		var treeLength = row.children.length;
		var nodeId = row.id;
		var pId = row.pid;
		var exclude = "Dashboard,首页";
		if(row.text == "首页"||row.text == "Dashboard"){
			return "<span nodeId = "+nodeId+" pId = "+pId+"  class='tree-checkbox tree-checkbox1 treeOper'></span>"+value;
		}else if(treeLength){ // 目录
			return "<span nodeId = "+nodeId+" pId = "+pId+" onclick='accessDir(\"sysRoleSelectMes\",event,this,\""+rowText+"\",\""+exclude+"\")'  class='tree-checkbox tree-checkbox0 treeOper tree-dir'></span>"+value;
		}else{ // 叶子节点
			return "<span nodeId = "+nodeId+" pId = "+pId+" onclick='accessTree(\"sysRoleSelectMes\",event,this,\""+rowText+"\",\""+exclude+"\")' class='tree-checkbox tree-checkbox0 treeOper'></span>"+value;
		}
	}
	function sysRoleCheckboxToggle(e,ele,rowId){
		e.stopPropagation();
		var treeData = $('#sysAddFeatureLimit').treegrid("getData");
		var leafLength = $('#sysAddFeatureLimit').treegrid("getChildren",rowId).length;
		
		//未勾选状态
		if($(ele).hasClass('tree-checkbox0') || $(ele).hasClass('tree-checkbox2')){
			$(ele).removeClass('tree-checkbox0').removeClass('tree-checkbox2').addClass('tree-checkbox1');
			//$($(ele).closest("td").prev()[0]).find(".treeOper").removeClass("tree-checkbox1").addClass("tree-checkbox0").trigger("click");
		}else{
			//取消勾选状态
			$(ele).removeClass('tree-checkbox1').addClass('tree-checkbox0');
		}
		updateWritableStatus(rowId);
	}
	// 处理目录级别的联动更新
	function accessDir(idName,e,node,rowText,excludes){ 
		$('#'+idName).css('visibility','hidden');
		e.stopPropagation();
	    var checkedClass = 'tree-checkbox1',
	        middleClass = 'tree-checkbox2',
	        unCheckClass = 'tree-checkbox0',
	        nodeId = $(node).attr('nodeid');
	    
	    var status = Array.from(node.classList).includes(checkedClass);

	    if(status) accessClass(node,unCheckClass,excludes);
	    else accessClass(node,checkedClass,excludes);
	    
	    cascade(nodeId);
	    
	 	// 级联批量只读
		function cascade(id){
			var nodeEl = $('[nodeid='+id+']')[0],
				readEl = $('[readid='+id+']')[0],
				writeEl = $('[writeid='+id+']')[0],
				nodeChecked = Array.from(nodeEl.classList).includes(checkedClass);

			var clsList = Array.from(readEl.classList),
				wList = Array.from(writeEl.classList),
				readChecked = clsList.includes(checkedClass),
				writeChecked = wList.includes(checkedClass);
			if(nodeChecked){ 
				if(writeChecked) {
					if(!readChecked) {
						readEl.click();
					}
				}else{
					writeEl.click();
				}
			}else{
				if(readChecked) {
					readEl.click();
				}
			}
		}
		function accessClass(node,cls,excludes){
	    	if(excludes) excludes = excludes.split(',');
	    	else excludes = [];
	        if(node) {
	          var classes = [checkedClass,middleClass,unCheckClass];
	          classes.map(function(item){
	            if(item == cls) node.classList.add(item);
	            else node.classList.remove(item);
	          });
	        }
	    }
	}
	function updateWritableStatus(nodeId) {
		var $el = $('[nodeid='+nodeId+']'),
			pid = $el.attr('pid'),
			node = $('[writeid='+nodeId+']')[0],
			pckbox = $('[writeid='+pid+']')[0],
			childs = $('[writepid='+pid+']'),
			checkedClass = 'tree-checkbox1',
		    middleClass = 'tree-checkbox2',
		    unCheckClass = 'tree-checkbox0',
		    excludes = '',
		    status = node?Array.from(node.classList).includes(checkedClass):true;

		downAccess(nodeId,status,excludes);
		upAccess(pid);
		cascade(nodeId);
		
		// 级联批量只读
		function cascade(id){
			var nodeEl = $('[nodeid='+id+']')[0],
				readEl = $('[readid='+id+']')[0],
				writeEl = $('[writeid='+id+']')[0],
				nodeChecked = Array.from(nodeEl.classList).includes(checkedClass);
			
			if(readEl){ // 是对批量操作的行
				var clsList = Array.from(readEl.classList),
					readChecked = clsList.includes(checkedClass),
					isReadMiddle = clsList.includes(middleClass);
				if(status) {
					if(!readChecked) {// 批量可写为选中，且批量只读为非选中
						readEl.click();
					}
					// 更新目录状态
					accessClass(nodeEl,checkedClass,excludes);
				}else {
					// 更新目录状态
					if(isReadMiddle || readChecked) {
						accessClass(nodeEl,middleClass,excludes);
					}else {
						accessClass(nodeEl,unCheckClass,excludes);
					}
				}
			}else{ // 对叶子节点的操作
				if(status) { // 叶子节点可写勾选时
					if(!nodeChecked) {
						nodeEl.click();
					}
				}else{// 取消叶子节点可写时
					var pTr = $(node).parents('tr:first'),
		            	readObj = pTr.find('.readOper')[0],
		            	readStatus = Array.from(readObj.classList).includes(checkedClass);
				
					if(readStatus) accessClass(nodeEl,middleClass,excludes);
				}
			}
		}
		// 向上递归
		function upAccess(pId){
			if(!pId) return;
			var isAll = true, has = false,
			    pckbox = $('[writeid='+pId+']:visible').get(0);
			if(pckbox) { // 遍历子节点，子节点的pid属性指向父级id
				var dirnode = $('[nodeid='+pId+']').get(0),
		    		rckbox = $('[readid='+pId+']').get(0);
			    $('[writepid='+pId+']').each(function(n,item){
			      var clist = Array.from(item.classList),
			          status = clist.includes(checkedClass);
			      
			      if(!status) isAll = false;
			      else has = true;
			      if(clist.includes(middleClass)) has = true;
			    });
			
			    if(isAll) {
			    	accessClass(pckbox,checkedClass,excludes);
			    	accessClass(rckbox,checkedClass,excludes);
			    }
			    else if(has) accessClass(pckbox,middleClass,excludes);
			    else accessClass(pckbox,unCheckClass,excludes);
			    
			    // 更新父级目录状态
			    var wStatus = Array.from(pckbox.classList).includes(checkedClass),
			    	rStatus = Array.from(rckbox.classList).includes(checkedClass),
			    	wmStatus = Array.from(pckbox.classList).includes(middleClass),
			    	rmStatus = Array.from(rckbox.classList).includes(middleClass);
			    if(wStatus && rStatus) {  // 批量只读和只写全为勾选时
			    	accessClass(dirnode,checkedClass,excludes);
			    }else if(wStatus || rStatus || wmStatus || rmStatus){
			    	accessClass(dirnode,middleClass,excludes);
			    }else { // 批量只读和只写全为不勾选时
			    	accessClass(dirnode,unCheckClass,excludes);
			    }
			    
			    // 递归处理父级节点状态: 当前节点可能是叶子或批量只读
			    var cpid = $(pckbox).attr('writepid');
			    upAccess(cpid);
			}
		}
		// 向下递归
		function downAccess(id,status,excludes){
			$('[writepid='+id+']').each(function(n,item){
			  //if(status) {accessClass(item,checkedClass,excludes);
			  //else accessClass(item,unCheckClass,excludes);
			  
			  var itemChecked = Array.from(item.classList).includes(checkedClass),
		  	  	  isVisible = $(item).is(':visible');
			  if(status != itemChecked) {
				  if(isVisible) item.click();
				  else if(!itemChecked){
					  item.click();
				  }
			  }
			  
			  downAccess($(item).attr('writeid'),status,excludes);
			});
		}
		/**
		* 设置状态的内部方法
		**/
		function accessClass(node,cls,excludes){
	    	if(excludes) excludes = excludes.split(',');
	    	else excludes = [];

	        if(node) {
	          var classes = [checkedClass,middleClass,unCheckClass];
	          classes.map(function(item){
	            if(item == cls) node.classList.add(item);
	            else node.classList.remove(item);
	          });
	        }
	    }
	}
	function sysSaveAddRole(){
		if(!$("#saveButtonRole").hasClass("forbidden")){
			var treeOper = $('#sysFeatureLimitDiv .tree-checkbox1.treeOper');
			var ptreeOper = $('#sysFeatureLimitDiv .tree-checkbox2.treeOper');
			var treeSource = $('#syseNBSourceDiv .tree-checkbox1');
			checkSysRoleName();
			if(isRolePass){
				if(treeOper.length == 0){
					$("#sysRoleSelectMes").css("visibility","visible");
					$("#sysAddRoleDiv").animate({
						scrollTop:200
					},500);
					return;
				}
				if(treeSource.length == 0){
					$("#sysAddSourceMes").css("visibility","visible");
					return;
				}
					 var check = $('#sysAddeNBSourceUl').tree('getChecked');
					 var checkStr = "";
					 check.map(function(item,index){
						 if(item.id == 0){
						 }else{
							 checkStr += item.id+",";
						 }
					 })
					 checkStr = checkStr.substring(0,checkStr.lastIndexOf(','));
					 var menuArr = [];
					  treeOper.each(function(index,item){
						 var length = $(item).parents("td:first").next().find("span.writeOper").length;
							 var obj = {};
							 var id = $(item).attr("nodeid");
							 if(id == "0"){
								 return;
							 }else{
								 obj.id = id;
							 }
							 var isWrite = $(item).parents("td:first").next().find("span.writeOper").hasClass("tree-checkbox1")
							 if(isWrite){
								 obj.write = true;
							 }else{
								 obj.write = false;
							 }
							 menuArr.push(obj);
					 })
					  ptreeOper.each(function(index,item){
				 		var obj = {};
				 		var id = $(item).attr("nodeid");
						 if(id == "0"){
					 		return;
				 		}else{
					 		obj.id = id;
					 		obj.write = false;
				 		}
				 		menuArr.push(obj)
			 		})
					 var params = {
						  "role_name":$("#sysAddRoleNameInput").val(),
						  "role_desc":$("#sysAddRoleTextare").val(),
						  "menu_ids":menuArr,
						  "device_group_ids":checkStr,
						  "operator_code":operator_code,
						  "type":"add"
					 }
					  params = JSON.stringify(params);
					  $.post("${ctx}/sys/role/save.action",{"params" : params},function(data){
						  if(data["success"]){
							  $("#sysRoleSetTable").datagrid("reload");
							  $("#saveButtonRole").addClass("forbidden");
								$('.success').fadeIn(300,function(){
									var  time = setTimeout(function(){
										cancelSysAddRole();
									},1500);
								})
						  }
					  },'json')
			}
		}
	}
	function checkSysRoleName(){
		var roleName = $("#sysAddRoleNameInput").val().trim();
		var roleLength = roleName.length;
		if(roleName == ""){
			$("#sysAddRoleName").css("visibility","visible");
			$("#sysAddRoleNameInput").css("border","1px solid #CC0000");
			$("#sysAddRoleName").html("<%=rb.getString("QingShuRuJueSeMingCheng")%>")
			$("#sysAddRoleDiv").animate({
				scrollTop:0
			},500);
			isRolePass = false;
			return;
		}else if(roleLength > 200){
			$("#sysAddRoleName").css("visibility","visible");
			$("#sysAddRoleNameInput").css("border","1px solid #CC0000");
			$("#sysAddRoleName").html("<%=rb.getString("ChangDuChaoChuFanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiDaChangDu")%><%=rb.getString("MaoHao")%>200")
			$("#sysAddRoleDiv").animate({
				scrollTop:0
			},500);
			isRolePass = false;
			return;
		}else{
			var params = {
				roleName : roleName
			}
			$.post("${ctx}/sys/role/checkRoleName.action",params,function(data){
				if(!data["success"]){
					$("#sysAddRoleName").html("<%=rb.getString("JueSeJiMingChengYiCunZai")%>")
					$("#sysAddRoleName").css("visibility","visible");
					$("#sysAddRoleNameInput").css("border","1px solid #CC0000");
					$("#sysAddRoleDiv").animate({
						scrollTop:0
					},500);
					isRolePass = false;
					return;
				}else{
					$("#sysAddRoleName").css("visibility","hidden");
					$("#sysAddRoleNameInput").css("border","1px solid #85A8BF");
					isRolePass = true;
					return;
				}
			},'json')
		}
	}
	function filterSysAddRole(){
		var deviceGroupName = $('#addDeviceName').val();
		$('#sysAddeNBSourceUl').tree("doFilter",deviceGroupName);
	}
	
</script>

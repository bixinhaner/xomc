<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
	#sysModifyFeatureLimitDiv .datagrid-row-over > td {
  		background: #fff !important;
  		cursor:pointer;
	}
	#sysModifySourceDiv .tree-node {
	  background: #fff !important;
	  cursor:pointer;
	  padding:3px 0px 3x 0px;
	}
	.errorMesRole {
		visibility:hidden;
		color:#CC0000;
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
			<label><%=rb.getString("YongHuZuMingCheng")%><%=rb.getString("MaoHao")%></label>
			<input id='sysModifyRoleNameInput' readonly style='padding-left:10px;width:390px;'  type='text'/>
		</p>
		<p style='margin-top:20px'>
			<label><%=rb.getString("MiaoShu")%><%=rb.getString("MaoHao")%></label>
			<textarea id='sysModifyRoleNameTextarea' style="resize:none;padding-left:10px;padding-top:5px;margin-top:5px;width:390px;height:125px;"></textarea>
		</p>
	</div>
	<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
		<ul class="omcPageTitleContainer_contain" style="width:80%">
			<li class="active"><%=rb.getString("GongNengQuanXianLieBiao")%></li>
		</ul>
	</div>
	<div class="deviceLogContainer" style='margin-top:15px;'>
		<p class='errorMesRole' id='sysModifyFeatureMes'><%=rb.getString("QingXuanZeGongNengQuanXian")%></p>
		<div id='sysModifyFeatureLimitDiv' class="tree-lines" style='user-select:none;height:450px;width:787px;border:1px solid #CCE1EF;margin-top:10px;overflow:auto'>
			<table style='height:100%;' id='sysModifyFeatureLimit'></table>
		</div>
	</div>
	<div class="omcPageTitleDiv_contain" style='margin-top:20px;'>
		<ul class="omcPageTitleContainer_contain" style="width:80%">
			<li class="active"><%=rb.getString("ZiYuanQuanXianLieBiao")%></li>
		</ul>
	</div>
	<div class="deviceLogContainer" style='margin-top:15px;'>
		<p class='errorMesRole' id='sysModifySourceMes'><%=rb.getString("QingXuanZeZiYuanQuanXian")%></p>
		<div class="queryGroup">
			<input id='sysModifyDeviceGroupName' style="width:450px;" placeholder="<%=rb.getString("Title_SheBeiMingCheng")%>" />
			<b class="el-icon el-icon-common-search" onclick="filterSysModifyRole()"></b>
		</div>
		<div id='sysModifySourceDiv' class="tree-lines" style='padding-left:10px;padding-top:10px;user-select:none;height:440px;width:777px;border:1px solid #CCE1EF;margin-top:10px;overflow:auto'>
			<ul id='sysModifySourceUl'></ul>
		</div>
	</div>
	<div class="windowButtonGroup" style="float:left !important;margin:30px 0 20px 58px">			
		<a class="linkbutton linkbutton_trend" onclick='sysModifySaveRole()'><span><%=rb.getString("QueDing")%></span></a>
		<a class="linkbutton linkbutton_nowanna" onclick='cancelSysModifyRole();'><span><%=rb.getString("QuXiao")%></span></a>
	</div>
	<div class='success'><%=rb.getString("XiuGaiJueSeChengGong")%></div>
</div>
<script>
	var select = $('#sysRoleSetTable').datagrid('getSelections')[0];
	if($("#curr_operator_span").text() == ""){
		var operator_code = "default";
	}else{
		var operator_code = operator_code;
	}
	$(function(){
		$('#sysModifyRoleNameInput').val(normalize(select.role_name));
		$('#sysModifyRoleNameTextarea').val(select.role_desc);
	    $('#sysModifyFeatureLimit').treegrid({
	        url:"${ctx}/sys/role/getFeature.action",
	        queryParams:{
	        	"role_id":select.role_id,
				"type":"modify",
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
	                formatter:sysModifyTreeMenuFormat
	            },
	            {
	                title: '<%=rb.getString("GongNengCaoZuo")%>',
	                field: 'operation',
	                width: 50,
	                formatter: sysModifyOnlyReadAndWriteFormat
	            }
	        ]],
	        onLoadSuccess:function(node,data){
	        	$("#sysModifyFeatureLimitDiv table .datagrid-btable:first>tbody>tr>td>div>.tree-file").removeClass("tree-file").addClass("tree-folder");
	        	var treetable = $(this).treegrid('getPanel').find('div.datagrid-body').parent().parent().parent();
	        	treetable.css('border-width','0px');
	        	var tr = $(this).treegrid('getPanel').find('div.datagrid-body tr.datagrid-row');
	        	 tr.each(function(){
	        		var td = $(this).children('td');
	        		td.css({
	        			'border-width':'0px'
	        		})
	        	});
	        	 var rootId = $('#sysModifyFeatureLimit').treegrid('getRoot').id;
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
	        	$('#sysModifyFeatureLimit').treegrid("toggle", row.id);
	        } ,
	        onExpand:function(row){
	      		var root = $('#sysModifyFeatureLimit').treegrid('getRoot');
	      		if(row.id!=root.id){
	      			var children = $('#sysModifyFeatureLimit').treegrid('getRoot').children;
	      			children.map(function(item,index){
	      				if(item.id == row.id){
	      					$('#sysModifyFeatureLimit').treegrid("expand", item.id);
	      				}else{
	      					$('#sysModifyFeatureLimit').treegrid("collapse", item.id);
	      				}
	      			})
	      		}
	      	}
	     });
	    $('#sysModifySourceUl').tree({
			url:"${ctx}/sys/role/getDeviceGroup.action",
			queryParams:{
				"role_id":select.role_id,
				"type":"modify",
				"group_name":$('#sysModifyGroupName').val(),
				"operator_code":operator_code
	        },
			checkbox:true,
			idField: 'id',            //定义关键字段来标识树节点。也就是数据的id
			treeField: 'text',        //定义树形显示字段
			fitColumns:true,
			animate:true,
			loadFilter: proccessData,
			onLoadSuccess:function(node,data){
				recurveFn(data);
				var root = $('#sysModifySourceUl').tree("getRoots");
				var children = $('#sysModifySourceUl').tree("getChildren",root);
				children.map(function(item,index){
					var isLeaf = $('#sysModifySourceUl').tree("isLeaf",item.target);
					if(!isLeaf && index == 1 || index == 0){
						$('#sysModifySourceUl').tree("expand",item.target);
					}else{
						$('#sysModifySourceUl').tree("collapse",item.target);
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
				$('#sysModifySourceUl').tree("toggle", node.target);
			},
			onCheck:function(){
				 $("#sysModifySourceMes").css("visibility","hidden"); 
			},
			onExpand:function(node){
				var nodeAll = $('#sysModifySourceUl').tree("getRoots")[0].children;
				nodeAll.map(function(item,index){
					if(node.id == item.id){
							$('#sysModifySourceUl').tree("expand",node.target);
					}else{
						$('#sysModifySourceUl').tree("collapse",item.target);
					}
				})
			}
			});
	    $("#sysModifyDeviceGroupName").bind("keyup",function(e){
	  		  if(e.keyCode == 13){
	  			filterSysModifyRole();
	  		  }
	  	  })
	})
		function sysModifyOnlyReadAndWriteFormat(value,row,rowIndex){
		var rowId = row.id,
			pId = row.pid,
			rowText = row.text,
			exclude = "Dashboard,首页";
		if(row.children && row.children.length>0){
			//return "";
			return ["<span readid = "+rowId+" readpid = "+pId+" onclick = 'accessTree(\"sysModifyFeatureMes\",event,this,\""+rowText+"\",\""+exclude+"\")'  class='tree-checkbox tree-checkbox0 readOper'></span>",
				    "<span class='tree-title'><%=rb.getString("ZhiDuQuanXuan")%></span>",
				    "<span writeid = "+rowId+" writepid = "+pId+" style='cursor:pointer;margin-left:5px;' onclick='sysModifyRoleCheckboxToggle(event,this,"+rowId+")' class='tree-checkbox tree-checkbox0 writeOper'></span>",
				    "<span class='tree-title'><%=rb.getString("KeXieQuanXuan")%></span>"].join("");
		}else{
	 	    if(row.text == "Dashboard"||row.text == "首页"){
				return "<span style='display:none;cursor:pointer;margin-left:10px;' class='tree-checkbox tree-checkbox1 readOper '></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span style='cursor:pointer;margin-left:5px;display:none' class='tree-checkbox tree-checkbox1 writeOper'></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiXie")%></span>";
			}else{
				if(row.checked){
					if(row.text == "EPC"||row.text == "告警确认"||row.text == "Alarm Confirm"||row.text == "告警清除"||row.text == "Clear Alarm"||row.text == "告警恢复"||row.text == "Restore Alarm"||row.text == "告警删除"||row.text == "Delete Alarm"||row.text == "用户向导"||row.text == "User Guide"||row.text == "关于"||row.text == "About"
						||row.text == "同步"||row.text == "Synchronize"||row.text == "关联的CPEs"||row.text == "CPEs"
							||row.text == "基站IMSI信息清除"||row.text == "Clear IMSI"||row.text == "系统资源监控"||row.text == "Resource"||row.text == "设置"||row.text == "Setting"||row.text == "激活"||row.text == "Active"||row.text == "eGW"||row.text == "网关"||row.text == "UPS"||row.text == "APN"||row.text == "快速配置"||row.text == "Quick Configuration"){
						return "<span style='display:none;cursor:pointer;margin-left:10px;' readviewpid='"+pId+"' class='tree-checkbox tree-checkbox1 readOper '></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span writeid="+rowId+" writepid="+pId+"  style='cursor:pointer;margin-left:5px;display:none' onclick='sysModifyRoleCheckboxToggle(event,this,"+rowId+")' class='tree-checkbox tree-checkbox1 writeOper'></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiXie")%></span>";
					}else{
						if(row.reWrite == "true"){
							if(row.write){
								return "<span style='cursor:pointer;margin-left:10px;' readviewpid='"+pId+"' class='tree-checkbox tree-checkbox1 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span writeid="+rowId+" writepid="+pId+"  style='cursor:pointer;margin-left:5px;' onclick='sysModifyRoleCheckboxToggle(event,this,"+rowId+")' class='tree-checkbox tree-checkbox1 writeOper'></span><span class='tree-title'><%=rb.getString("ZhiXie")%></span>"
							}else{
								return "<span style='cursor:pointer;margin-left:10px;' readviewpid='"+pId+"' class='tree-checkbox tree-checkbox1 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span writeid="+rowId+" writepid="+pId+"  style='cursor:pointer;margin-left:5px;' onclick='sysModifyRoleCheckboxToggle(event,this,"+rowId+")' class='tree-checkbox tree-checkbox0 writeOper'></span><span class='tree-title'><%=rb.getString("ZhiXie")%></span>"
							}
						}else{
							return "<span style='cursor:pointer;margin-left:10px;' readviewpid='"+pId+"' class='tree-checkbox tree-checkbox1 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>"
						}
					}
				}else{
					if(row.text == "EPC"||row.text == "告警确认"||row.text == "Alarm Confirm"||row.text == "告警清除"||row.text == "Clear Alarm"||row.text == "告警恢复"||row.text == "Restore Alarm"||row.text == "告警删除"||row.text == "Delete Alarm"||row.text == "用户向导"||row.text == "User Guide"||row.text == "关于"||row.text == "About"
						||row.text == "同步"||row.text == "Synchronize"||row.text == "关联的CPEs"||row.text == "CPEs"
							||row.text == "基站IMSI信息清除"||row.text == "Clear IMSI"||row.text == "系统资源监控"||row.text == "Resource"||row.text == "设置"||row.text == "Setting"||row.text == "激活"||row.text == "Active"||row.text == "eGW"||row.text == "网关"||row.text == "UPS"||row.text == "APN"||row.text == "快速配置"||row.text == "Quick Configuration"){
						return "<span style='display:none;cursor:pointer;margin-left:10px;' readviewpid='"+pId+"' class='tree-checkbox tree-checkbox0 readOper '></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span writeid="+rowId+" writepid="+pId+" style='cursor:pointer;margin-left:5px;display:none' onclick='sysModifyRoleCheckboxToggle(event,this,"+rowId+")' class='tree-checkbox tree-checkbox0 writeOper'></span><span style='display:none' class='tree-title'><%=rb.getString("ZhiXie")%></span>";
					}else{
						if(row.reWrite == "true"){
							return "<span style='cursor:pointer;margin-left:10px;' readviewpid='"+pId+"' class='tree-checkbox tree-checkbox0 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>&nbsp;&nbsp;<span writeid="+rowId+" writepid="+pId+"  style='cursor:pointer;margin-left:5px;' onclick='sysModifyRoleCheckboxToggle(event,this,"+rowId+")' class='tree-checkbox tree-checkbox0 writeOper'></span><span class='tree-title'><%=rb.getString("ZhiXie")%></span>"
						}else{
							return "<span style='cursor:pointer;margin-left:10px;' readviewpid='"+pId+"' class='tree-checkbox tree-checkbox0 readOper '></span><span class='tree-title'><%=rb.getString("ZhiDu")%></span>"
						}
					}
				}
			}
		}
	}
	function sysModifyTreeMenuFormat(value,row,rowIndex){
		var rowText = row.text;
		var treeLength = row.children.length;
		var nodeId = row.id;
		var pId = row.pid;
		var exclude = "Dashboard,首页";
		if(row.text == "Dashboard"||row.text == "首页"){
			return "<span nodeId = "+nodeId+" pId = "+pId+"   class='tree-checkbox tree-checkbox1 treeOper'></span>"+value;
		}else if(treeLength) {
			return "<span nodeId = "+nodeId+" pId = "+pId+" onclick = 'accessDir(\"sysModifyFeatureMes\",event,this,\""+rowText+"\",\""+exclude+"\")'  class='tree-checkbox tree-checkbox0 treeOper tree-dir'></span>"+value;
		}else{
			if(row.checked){
				if(row.reWrite == 'true') {
					if(row.write) return "<span nodeId = "+nodeId+" pId = "+pId+" onclick = 'accessTree(\"sysModifyFeatureMes\",event,this,\""+rowText+"\",\""+exclude+"\")'  class='tree-checkbox tree-checkbox1 treeOper'></span>"+value;
					else return "<span nodeId = "+nodeId+" pId = "+pId+" onclick = 'accessTree(\"sysModifyFeatureMes\",event,this,\""+rowText+"\",\""+exclude+"\")'  class='tree-checkbox tree-checkbox2 treeOper'></span>"+value;
				}else{
					return "<span nodeId = "+nodeId+" pId = "+pId+" onclick = 'accessTree(\"sysModifyFeatureMes\",event,this,\""+rowText+"\",\""+exclude+"\")'  class='tree-checkbox tree-checkbox1 treeOper'></span>"+value;
				}
			}else{
				return "<span nodeId = "+nodeId+" pId = "+pId+" onclick = 'accessTree(\"sysModifyFeatureMes\",event,this,\""+rowText+"\",\""+exclude+"\")'  class='tree-checkbox tree-checkbox0 treeOper'></span>"+value;
			}
		}
	}
	function sysModifyRoleCheckboxToggle(e,ele,rowId){
		e.stopPropagation();
		var treeData = $('#sysModifyFeatureLimit').treegrid("getData");
		var leafLength = $('#sysModifyFeatureLimit').treegrid("getChildren",rowId).length;
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
				}else {
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
	        if(node && !excludes.includes(node.nextSibling.nodeValue)) {
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
		    status = Array.from(node.classList).includes(checkedClass);

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

	        if(node && !excludes.includes(node.nextSibling.nodeValue)) {
	          var classes = [checkedClass,middleClass,unCheckClass];
	          classes.map(function(item){
	            if(item == cls) node.classList.add(item);
	            else node.classList.remove(item);
	          });
	        }
	    }
	}
	function sysModifySaveRole(){
		var treeOper = $('#sysModifyFeatureLimitDiv .tree-checkbox1.treeOper');
		var ptreeOper = $('#sysModifyFeatureLimitDiv .tree-checkbox2.treeOper');
		var treeSource = $('#sysModifySourceDiv .tree-checkbox1');
	 	if(treeOper.length == 0){
			$("#sysModifyFeatureMes").css("visibility","visible");
			$("#sysModifyRoleDiv").animate({
				scrollTop:200
			},500);
			return;
		}
		if(treeSource.length == 0){
			$("#sysModifySourceMes").css("visibility","visible");
			return;
		} 
		 var check = $('#sysModifySourceUl').tree('getChecked');
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
					 return
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
			  "role_id":select.role_id,
			  "role_desc":$("#sysModifyRoleNameTextarea").val(),
			  "menu_ids":menuArr,
			  "device_group_ids":checkStr,
			  "operator_code":operator_code,
			  "type":"modify"
		 }
		  params = JSON.stringify(params);
		  $.post("${ctx}/sys/role/save.action",{"params" : params},function(data){
			  if(data["success"]){
				  $("#sysRoleSetTable").datagrid("reload");
					$('.success').fadeIn(300,function(){
						var  time = setTimeout(function(){
							cancelSysModifyRole();
						},1500);
					})
			  }
		  },'json')
	}
	function filterSysModifyRole(){
		var deviceGroupName = $('#sysModifyDeviceGroupName').val();
		$('#sysModifySourceUl').tree("doFilter",deviceGroupName);
	}
	function recurveFn(data){
		data.map(function(item,index){
			if(item.checked){
				$('#sysModifySourceUl').tree("check",item.target);
			}
			if(item.children){
				recurveFn(item.children);
			}
		})
	}
	function normalize(str){
		var divDom = document.createElement('div');
		divDom.innerHTML = str;
		var text = divDom.innerText;
		delete divDom;
		return text;
	}
</script>

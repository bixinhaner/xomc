<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.detailMesDiv > div{
		display:inline-block;
	}
	#winModuleUpgradeFileInfo label{
		display:block;
		margin-bottom:5px;
	}
	#winModuleUpgradeFileInfo p{
		font-size:12px;
		color:#CC0000;
		margin-top:5px;
		margin-bottom:7px;
		visibility:hidden;
	}
	.detailMesDiv {
		display: flex;
		flex-wrap: wrap;
	}
	.detailMesDiv > div {
		flex: 1 1 45%;
	}
	.el-card__body{
		display:flex;
		flex-direction:column;
		flex:1 1 auto;
		height:100%;
		overflow:auto;
	}
	.couple-left{
		height:100%;
	}
	input.file_info {
		background:#FFF;
	}
	input.file_info:disabled {
		background:#F5F7FA;
	}
	.combobox-item-selected {
		background:#F2F6FF;
	}
	.combobox-item-selected::after {
		content:'\e734';
		float:right;
		margin-right:15px;
		font-family:'el-icon';
		font-size:14ppx;
		color:#4D84FF;
	}
	.combobox-item-hover {
		background:#F2F6FF;
	}
	.selectedDiv {
		display:inline-block;
		line-height:20px;
		padding:0 15px;
		margin-right:20px;
		border:1px solid #4D84FF;
	}
	.selectedDiv .el-icon-operation-delete {
		font-size:15px;
		margin-left:20px;
	}
	.coverInput {
		position:absolute;
		left:0;
		top:0;
		width:330px;
		height:24px;
		border:none;
	}
</style>
<div id="winModuleUpgradeFileInfo" style="display:flex;flex-direction:column;height:100%;">
		<div class='el-card__header'>
			<span><%=rb.getString("WenJianXinXi")%></span>
			<span class='el-icon el-icon-close' onclick='closeImportFileWindow()'></span>
		</div>
		<div class='el-card__body'>
			<div class='detailMesDiv'  style="background:#fff;padding:20px;">
			    <div class=''>
					<label for="moduleFilePath" style="width: 120px;"><%=rb.getString("WenJian")%><%=rb.getString("MaoHao")%></label>
					<input id="moduleFilePath" type="text" class="border border-box file_info" readonly="readonly" style="vertical-align:middle;padding-right:27px;width:350px;"/>
					<a class="el-icon el-icon-operation-import" title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="scanClick()" 
			    		style="vertical-align:middle; margin:0 2px 0 -29px;display:inline-block;width:23px;height:24px;line-height:24px;background-color:#fff;">
					</a>
					<p><%=rb.getString("QingXianXuanZeWenJian")%></p>
				</div>
			    <div style=''>
					<label for="moduleVersion" style="width: 120px;"><%=rb.getString("BanBen")%></label>
					<input onblur='checkVersion()' id="moduleVersion" type="text" class="border border-box file_info required" maxlength=45 style="width: 350px;height:26px;"/>
					<p><%=rb.getString("QingShuRuWenJianBanBen")%></p>
				</div>
				<div style="width:90%;flex:unset;">
					<label for="moduleName" style="width: 120px;"><%=rb.getString("MoKuaiMingCheng")%></label>
					<input id="moduleName" type="text" class="border border-box file_info required" maxlength=100 style="width: 350px;height:26px;" />
					<span class='el-icon el-icon-plus' onclick="addModule()"></span>
					<div class="selectedModule" style="margin-top:10px;"></div>
					<p><%=rb.getString("QingXuanZeMoKuaiXingHao")%></p>
				</div>
				
				<div style="width:90%;height:400px;flex:unset;">
					<label for="destVersionList_datagrid" style="width: 120px;"><%=rb.getString("MuBiaoBanBen")%></label>
		  			<div id="destVersionList_datagrid" ></div>
		  			<p class="errorText_oriVersion"><%=rb.getString("QingXianXuanZeWenJian")%></p>
				</div>
				<div style="margin-top:50px;" id="descEditBox">
					<label for="moduleDesc" style="width: 120px; vertical-align: top;"><%=rb.getString("MiaoShu")%></label>
					<textarea id="moduleDesc" cols="20" style="padding-top:5px;font-size: 12px;width: 766px; height: 150px; resize: none;" rows="5"
						maxlength=500 class="border-box border file_info"></textarea>
				</div>
			</div>
		</div>
		<div class="slideFooter" >
			<a id='upgradeSubmit' class="linkbutton linkbutton_trend" onclick='submitUploadForm()'><span><%=rb.getString("QueDing")%></span></a>
			<a class="linkbutton linkbutton_nowanna " onclick="closeImportFileWindow();"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
		
	<!-- 指标选择toolbar -->
	<div id="toolbar_destVersionList_datagrid" style="padding:5px 10px;">
		<div class="queryGroup" style="margin:0 0 0 20px;">
			<input name="destVersion" id="destVerQuery" style="width:300px;" placeholder="<%=rb.getString("MuBiaoBanBen")%>">
			<b class="el-icon el-icon-common-search" onclick="javascript: $('#destVersionList_datagrid').pairgrid('reload')"></b>
		</div>
	</div>
	
	<%-- 上传文件的用的表单 --%>
	<form enctype="multipart/form-data" method="post" id="uploadFileForm_moduleVersion">
	    <input name="newFileName" id="newFileName" value="" hidden="true">
	    <input name="fileSize" id="fileSize" value="" hidden="true">
	    <input name="version" value="" type="hidden"/>
	    <input name="destVersion" value="" type="hidden"/>
	    <input name="moduleName" value="" type="hidden"/>
	    <input name="desc" value="" type="hidden"/>
	    <input name="uploadFile" id="uploadFile_midVersion"  type="file" style="display: none;">
	    <input name="md5" value="" type="hidden"/>
	</form>
</div>
<script>
	var deviceTypeArr = [];
	var selText = '';
	var modelPageType = '${type}';
	$(function(){
		
		
		if (modelPageType == 'edit'){
			$("#moduleFilePath").attr("disabled","disabled");
			$("#moduleVersion").attr("disabled","disabled");
			$(".el-icon-operation-import").hide();
		}
		
		if (modelPageType == 'view'){
			$("#moduleFilePath").attr("disabled","disabled");
			$("#moduleVersion").attr("disabled","disabled");
			$("#moduleName").hide();
			$("#moduleDesc").attr("disabled","disabled");
			$(".el-icon-operation-import").hide();
			$(".slideFooter").hide();
			$(".el-icon-plus").hide();
			$("#realInput").hide();
			$('#destVersionList_datagrid').pairgrid({readonly:true});
		}
		
		if ( modelPageType != 'view') {
			$("#moduleName").combobox({
				multiple:true,
				url:'${ctx}/cell/version/queryModuleNames.action',
		    	textField:'moduleName',
		    	valueField:'moduleName',
		    	selectOnNavigation:false,
				onSelect: function(record){
					var selDiv = "<div class='selectedDiv'>" + record.moduleName + "<span class='el-icon el-icon-operation-delete' onclick='unselectModule(this)'></span>" +"</div>";
					$(".selectedModule").append(selDiv);
					var newText = $("#moduleName").combobox("getText");
					selText += newText;
					var newValue = $("#moduleName").combobox("getValues");
					$(".selectedModule").siblings("p").css("visibility","hidden");
					
				},
				onUnselect: function(record){
					$(".selectedDiv").each(function(){
						if ($(this).text() == record.moduleName){
							$(this).remove()
						}
					})
					
				},
				onLoadSuccess:function(){
					$("#moduleName").next(".textbox").append("<input id='realInput' class='coverInput' />");
					
					$("#realInput").keyup(function(ev){
						
						var nowText = $("#moduleName").combobox("getText");
						var addText = $("#realInput").val();
						$("#moduleName").combobox("setText",selText+','+addText);
						
						$("#moduleName").next().find('input.validatebox-text').trigger('query');
						
					})
				},
				filter:function(q,row){
					
					var opts = $(this).combobox('options');
					var realVal = $("#realInput").val();
					return row[opts.textField].indexOf(selText+','+realVal) == 0;
				}
				
				
			})
		}
		
		$("#uploadFile_midVersion").bind("change", function() {
			$("#winModuleUpgradeFileInfo #moduleFilePath").val(this.value);
			 var filePath = $("#uploadFile_midVersion").val();
			 var pathSplit = filePath.split(/\\/);
			 var filename = pathSplit[pathSplit.length - 1];
			 if(filename.length > 100){
				 $("#moduleFilePath").siblings("p").css("visibility","visible").html("<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>");
			 }else{
				 
				 if(fileFormatMatch(this.value,"tgz")){
					 $("#moduleFilePath").siblings("p").css("visibility","hidden");
					 $("#winModuleUpgradeFileInfo #moduleVersion").val(filename.substring(0,filename.lastIndexOf(".")));
					 checkVersion();
				 }else{
					 $("#moduleFilePath").siblings("p").css("visibility","visible").html("<%=rb.getString("ZhIZhiChiTGZWenJian")%>");
				 }
			 }
		});
		
		
		$('#destVersionList_datagrid').pairgrid({
	        idField: 'version',
	    	leftUrl : '${ctx}/cell/version/queryCpeDestVersion.action',
	    	rightData:[],
	        border:false,
	        fit:true,
	        model:"normal",
	        fitColumns:true,
	        rownumbers : true,
	        striped : true,
	        singleSelect : true,
			pageList : [ 50, 100, 150, 200, 250, 300 ],
	        pagination: true,
	        pagePosition: 'bottom',
			zone : [50,50],
			queryName : 'version',
			messages:{queryName:'<%=rb.getString("MuBiaoBanBen")%>'},
	        toolBar:"#toolbar_destVersionList_datagrid",
	        onCheck: destVerDatagridCheck,
	        onLoadSuccess : datagridLoadSuccess,
	        leftBeforeLoad : beforeLoad_gridCell_destVer_select_left,
	        rightBeforeLoad : beforeLoad_gridCell_destVer_select_right,
	  	    leftColumns : [{
							field : 'ck',
							checkbox:true,
						},{	
			            	field:"version",
			    			width : 100,
			    			title: '<%=rb.getString("MuBiaoBanBen")%>'
			            }],
			rightColumns : [{
			                field : 'version',
			                title : '<%=rb.getString("MuBiaoBanBen")%>',
			                width : 100,
				        }]
	  	});
		
		
	})
	
	//取消或删除已选择的module name
	function unselectModule(ele){
		var json =$("#moduleName").combobox('getData');
		var thisModel = $(ele).parent('.selectedDiv').text();
		$.each(json,function(i){
			if (thisModel == this.moduleName){
				$("#moduleName").combobox('unselect',this.moduleName)
			}
		})
		$(ele).parent('.selectedDiv').remove();
	}
	
	function beforeLoad_gridCell_destVer_select_left(param){
		param.version = $("#destVerQuery").val();
	}
	function beforeLoad_gridCell_destVer_select_right(param){
		param.limitNum = "9";
	}
	
	function destVerDatagridCheck(){
		$(".errorText_oriVersion").css("visibility","hidden");
	}
	
	// 打开窗口，选择文件
	function scanClick() {
		$('#uploadFile_midVersion').click();
	}
	
	function checkVersion(){
		if($("#moduleVersion").val() == ""){
			$("#moduleVersion").siblings("p").css("visibility","visible");
			return false;
		}else{
			$("#moduleVersion").siblings("p").css("visibility","hidden");
			return true;
		}
	}
	
	function addModule(){
		var realVal = $("#realInput").val();
		if (realVal.length > 0){
			var selDiv = "<div class='selectedDiv'>" + realVal + "<span class='el-icon el-icon-operation-delete' onclick='unselectModule(this)'></span>" +"</div>";
			$(".selectedModule").append(selDiv);
			$("#realInput").val('');
			$(".selectedModule").siblings("p").css("visibility","hidden");
		}else{
			
		}
		
		
		
	}
	
	
	function submitUploadForm(){
		if(!$("#upgradeSubmit").hasClass("forbidden")){
			if($("#moduleFilePath").val() == ""){
				$("#moduleFilePath").siblings("p").css("visibility","visible").html("<%=rb.getString("QingXianXuanZeWenJian")%>");
			}
			checkVersion();
			
			var isSubmitFlag = true;
			
			//选中的指标
			var selectedDestVersArr = ''; 
			var selectedDestVers = $("#destVersionList_datagrid").pairgrid("getData");
			if(selectedDestVers.length>0){
				$(".errorText_oriVersion").css("visibility","hidden");
				$.each(selectedDestVers,function(index,ele){
					selectedDestVersArr = selectedDestVersArr + ele.version + ',';
				})
				selectedDestVersArr = selectedDestVersArr.substring(0,selectedDestVersArr.length-1)
			}else {
				$(".errorText_oriVersion").css("visibility","visible");
			}
			var selectedName = '';
			if ($(".selectedDiv").length > 0){
				$(".selectedDiv").each(function(){
					selectedName = selectedName + $(this).text() +',';
				})
				selectedName = selectedName ? selectedName.substring(0,selectedName.length-1):selectedName;
			}else {
				$(".selectedModule").siblings("p").css("visibility","visible");
			}
			
			
			$("#winModuleUpgradeFileInfo p").map(function(index,item){
				if($(item).css("visibility") == "visible"){
					isSubmitFlag = false;
				}
			})
			
			 var grid;
				    grid = $("#fileinfo_cpemodule");
			
			if(isSubmitFlag){
				if (modelPageType == 'add'){
					var files = document.getElementById("uploadFile_midVersion").files;
				    var filePath = $("#uploadFile_midVersion").val();
					
				    var pathSplit = filePath.split(/\\/);
				    var filename = pathSplit[pathSplit.length - 1];
				    // 向form中赋值
				    $("#uploadFileForm_moduleVersion [name=fileSize]").val(files[0].size);
				    $("#uploadFileForm_moduleVersion [name=newFileName]").val(filename);
				    $("#uploadFileForm_moduleVersion [name=version]").val($("#winModuleUpgradeFileInfo #moduleVersion").val());
				    $("#uploadFileForm_moduleVersion [name=destVersion]").val(selectedDestVersArr);
				    $("#uploadFileForm_moduleVersion [name=moduleName]").val(selectedName);
				    $("#uploadFileForm_moduleVersion [name=desc]").val($("#winModuleUpgradeFileInfo #moduleDesc").val().replace(/\n/g, " "));
				    $('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
				    $("#winUploadPro").window("open");// 打开进度条窗口
				    intervalGetProgress = window.setInterval(function(){
				    	getUploadProgress();// 定时读取进度
						var dom = $("#progressUploadFile");
						if(dom.length == 0) clearInterval(intervalGetProgress);
				    }, 1000);
				    
				    uploadWithProgress({
				    	url: "${ctx}/cell/version/uploadModuleVersionFile.action",
				    	form: document.querySelector("#uploadFileForm_moduleVersion"),
				    	success: function (data) {
				        	// 清除定时器
				            window.clearInterval(intervalGetProgress);
				            $("#winUploadPro").window("close");// 关闭进度条窗口
				            if (data["MD5"]) {
				            	$("#upgradeSubmit").addClass("forbidden");
				            	document.getElementById("uploadFile_midVersion").value = "";// 置空文件组件
				            	$.messager.alert(TiShi,"<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>"+data["MD5"]);
				            	closeImportFileWindow();
				            	grid.datagrid("reload");
				            } else {
				            	showMsg('error_msg',data["message"]);
				            }
				        }
				    })
				}
				
				if (modelPageType == 'edit'){
					
					var params = { 
							moduleVersion : $("#winModuleUpgradeFileInfo #moduleVersion").val(),
							moduleName:selectedName,
							destVersion:selectedDestVersArr, 
							desc:$("#winModuleUpgradeFileInfo #moduleDesc").val().replace(/\n/g, " ")
					};
				    $.post("${ctx}/cell/version/updateModuleVersionInfo.action", params, function(data) {
				        if (data["success"]) {
							showMsg("success_msg",'<%=rb.getString("ChengGong")%>')
							$('.editFileUpgradeDiv').animate({right:'-1000px'},400,function(){
								$('.editFileUpgradeDiv').html("");
								grid.datagrid("reload");
							});
				        }else{
				        	showMsg('error_msg',data["message"]);
				        	$('.editFileUpgradeDiv').animate({right:'-1000px'},400,function(){
								$('.editFileUpgradeDiv').html("");
							});
				        }
				    }, "json");
					
				}
				
				
			}
		}
	}
</script>
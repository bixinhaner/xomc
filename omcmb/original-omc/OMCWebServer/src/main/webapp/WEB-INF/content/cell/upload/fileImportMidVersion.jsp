<%@ page contentType="text/html;charset=UTF-8"%>
<%@ include file="/common/taglibs.jsp" %>

<style>
	.detailMesDiv > div{
		display:inline-block;
	}
	#winMidUpgradeFileInfo label{
		display:block;
		margin-bottom:5px;
	}
	#winMidUpgradeFileInfo p{
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
</style>
<div id="winMidUpgradeFileInfo" style="display:flex;flex-direction:column;height:100%;">
		<div class='el-card__header'>
			<span><%=rb.getString("WenJianXinXi")%></span>
			<span class='el-icon el-icon-close' onclick='closeImportFileWindow()'></span>
		</div>
		<div class='el-card__body'>
			<div class='detailMesDiv'  style="background:#fff;padding:20px;">
			    <div class=''>
					<label for="midFilePath" style="width: 120px;"><%=rb.getString("WenJian")%><%=rb.getString("MaoHao")%></label>
					<input id="midFilePath" type="text" class="border border-box file_info" readonly="readonly" style="vertical-align:middle;padding-right:27px;width:350px;"/>
					<a class="el-icon el-icon-operation-import" title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="scanClick()" 
			    		style="vertical-align:middle; margin:0 2px 0 -29px;display:inline-block;width:23px;height:24px;line-height:24px;background-color:#fff;">
					</a>
					<p><%=rb.getString("QingXianXuanZeWenJian")%></p>
				</div>
			    <div style=''>
					<label for="midVersion" style="width: 120px;"><%=rb.getString("BanBen")%></label>
					<input onblur='checkVersion()' id="midVersion" type="text" class="border border-box file_info required" maxlength=45 style="width: 350px;height:26px;"/>
					<span class='operationDiv operation_getFocus fileEditStar'></span>
					<p><%=rb.getString("QingShuRuWenJianBanBen")%></p>
				</div>
				
				<div style="width:90%;flex:unset;">
					<label for="modelName" style="width: 120px;"><%=rb.getString("ChanPinXingHao")%></label>
					<input id="modelName" type="text" class="border border-box file_info required" maxlength=100 style="width: 350px;height:26px;" />
					<p class="errorText_modelName"><%=rb.getString("QingXuanZeChanPinXingHao")%></p>
				</div>
				<div style="width:90%;height:400px;flex:unset;">
					<label for="oriVersionList_datagrid" style="width: 120px;"><%=rb.getString("ChuShiBanBen")%></label>
		  			<div id="oriVersionList_datagrid" ></div>
		  			<p class="errorText_oriVersion"><%=rb.getString("QingXianXuanZeWenJian")%></p>
		  			<!-- <input id="selectTableValue" style="display:none;" /> -->
				</div>
				<div style="margin-top:50px;" id="descEditBox">
					<label for="midDesc" style="width: 120px; vertical-align: top;"><%=rb.getString("MiaoShu")%></label>
					<textarea id="midDesc" cols="20" style="padding-top:5px;font-size: 12px;width: 766px; height: 150px; resize: none;" rows="5"
						maxlength=500 class="border-box border file_info"></textarea>
				</div>
			</div>
		</div>
		<div class="slideFooter" >
			<a id='upgradeSubmit' class="linkbutton linkbutton_trend" onclick='submitUploadForm()'><span><%=rb.getString("QueDing")%></span></a>
			<a class="linkbutton linkbutton_nowanna " onclick="closeImportFileWindow();"><span><%=rb.getString("QuXiao")%></span></a>
		</div>
		
	<!-- 指标选择toolbar -->
	<div id="toolbar_oriVersionList_datagrid" style="padding:5px 10px;">
		<div class="queryGroup" style="margin:0 0 0 20px;">
			<input name="version" id="queryOriVersion" style="width:300px;" placeholder="<%=rb.getString("BanBen")%> ">
			<b class="el-icon el-icon-common-search" onclick="javascript: $('#oriVersionList_datagrid').pairgrid('reload')"></b>
		</div>
	</div>
	
	<%-- 上传文件的用的表单 --%>
	<form enctype="multipart/form-data" method="post" id="uploadFileForm_midVersion">
	    <input name="newFileName" id="newFileName" value="" hidden="true">
	    <input name="fileSize" id="fileSize" value="" hidden="true">
	    <input name="version" value="" type="hidden"/>
	    <input name="modelName" value="" type="hidden"/>
	    <input name="oriVersion" value="" type="hidden"/>
	    <input name="desc" value="" type="hidden"/>
	    <input name="uploadFile" id="uploadFile_midVersion"  type="file" style="display: none;">
	    <input name="md5" value="" type="hidden"/>
	</form>
</div>
<script>
	var deviceTypeArr = [];
	var midPageType = '${type}';
	$(function(){
		
		if (midPageType == 'edit'){
			$("#midFilePath").attr("disabled","disabled");
			$("#midVersion").attr("disabled","disabled");
			$(".el-icon-operation-import").hide();
		}
		if (midPageType == 'view'){
			$("#midFilePath").attr("disabled","disabled");
			$("#midVersion").attr("disabled","disabled");
			$("#midDesc").attr("disabled","disabled");
			$(".slideFooter").hide();
			$('#oriVersionList_datagrid').pairgrid({readonly:true});
			$(".el-icon-operation-import").hide();
		}
		
		
			$("#modelName").combobox({
				url:'${ctx}/cell/CPE/queryModelNames.action',
		    	textField:'model_name',
		    	valueField:'model_name',
				onLoadSuccess:function(){
					if (midPageType != 'add'){
						$("#modelName").combobox("disable");
					}
				},
				onSelect:function(){
					$(".errorText_modelName").css("visibility","hidden");
				}
			})
		
		$("#uploadFile_midVersion").bind("change", function() {
			$("#winMidUpgradeFileInfo #midFilePath").val(this.value);
			 var filePath = $("#uploadFile_midVersion").val();
			 var pathSplit = filePath.split(/\\/);
			 var filename = pathSplit[pathSplit.length - 1];
			 if(filename.length > 100){
				 $("#midFilePath").siblings("p").css("visibility","visible").html("<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>");
			 }else{
				 
				 if(fileFormatMatch(this.value,"bin,tgz")){
					 $("#midFilePath").siblings("p").css("visibility","hidden");
					 $("#winMidUpgradeFileInfo #midVersion").val(filename.substring(0,filename.lastIndexOf(".")));
					 checkVersion();
				 }else{
					 $("#midFilePath").siblings("p").css("visibility","visible").html("<%=rb.getString("ZhIZhiChiTGZBINWenJian")%>");
				 }
			 }
		});
		
		$('#oriVersionList_datagrid').pairgrid({
	        idField: 'software_version',
	    	leftUrl : '${ctx}/cell/CPE/queryCpeCurrVersions.action',
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
			messages:{queryName:'<%=rb.getString("BanBen")%>'},
	        toolBar:"#toolbar_oriVersionList_datagrid",
	        leftBeforeLoad : beforeLoad_oriVersionList_datagrid_left,
	        rightBeforeLoad : beforeLoad_oriVersionList_datagrid_right,
	        onCheck: oriVersionCheck,
	        onLoadSuccess : datagridLoadSuccess,
	  	    leftColumns : [{
							field : 'ck',
							checkbox:true,
						},{	
			            	field:"software_version",
			    			width : 100,
			    			title: '<%=rb.getString("ChuShiBanBen")%>'
			            }],
			rightColumns : [{
			                field : 'software_version',
			                title : '<%=rb.getString("ChuShiBanBen")%>',
			                width : 100,
				        }]
	  	});
		
		
	})
	
	function oriVersionCheck(){
		$(".errorText_oriVersion").css("visibility","hidden");
	}
	
	function beforeLoad_oriVersionList_datagrid_left(param){
		param.version = $("#queryOriVersion").val();
	}
	function beforeLoad_oriVersionList_datagrid_right(param){
		param.limitNum = "9";
	}
	
	// 打开窗口，选择文件
	function scanClick() {
		$('#uploadFile_midVersion').click();
	}
	
	function checkVersion(){
		if($("#midVersion").val() == ""){
			$("#midVersion").siblings("p").css("visibility","visible");
			return false;
		}else{
			$("#midVersion").siblings("p").css("visibility","hidden");
			return true;
		}
	}
	
	
	function submitUploadForm(){
		if(!$("#upgradeSubmit").hasClass("forbidden")){
			if($("#midFilePath").val() == ""){
				$("#midFilePath").siblings("p").css("visibility","visible").html("<%=rb.getString("QingXianXuanZeWenJian")%>");
			}
			checkVersion();
			
			var isSubmitFlag = true;
			
			//选中的指标
			var selectedVersionArr = ''; 
			var selectedVersions = $("#oriVersionList_datagrid").pairgrid("getData");
			if(selectedVersions.length>0){
				$(".errorText_oriVersion").css("visibility","hidden");
				$.each(selectedVersions,function(index,ele){
					selectedVersionArr = selectedVersionArr + ele.software_version + ',';
				})
				selectedVersionArr = selectedVersionArr.substring(0,selectedVersionArr.length-1)
			}else {
				$(".errorText_oriVersion").css("visibility","visible");
			}
			
			var modelName = $("#modelName").combobox("getValue");
			if ( modelName){
				$(".errorText_modelName").css("visibility","hidden");
			}else{
				$(".errorText_modelName").css("visibility","visible");
			}
			
			$("#winMidUpgradeFileInfo p").map(function(index,item){
				if($(item).css("visibility") == "visible"){
					isSubmitFlag = false;
				}
			})
			
			var midVersion = $("#winMidUpgradeFileInfo #midVersion").val();
			var midDesc = $("#winMidUpgradeFileInfo #midDesc").val().replace(/\n/g, " ");
			var grid;
		    	grid = $("#fileinfo_cpemidversion");
			if(isSubmitFlag){
				if (midPageType == 'add'){
					var files = document.getElementById("uploadFile_midVersion").files;
				    var filePath = $("#uploadFile_midVersion").val();
					
				    var pathSplit = filePath.split(/\\/);
				    var filename = pathSplit[pathSplit.length - 1];
				    // 向form中赋值
				    $("#uploadFileForm_midVersion [name=fileSize]").val(files[0].size);
				    $("#uploadFileForm_midVersion [name=newFileName]").val(filename);
				    $("#uploadFileForm_midVersion [name=version]").val(midVersion);
				    $("#uploadFileForm_midVersion [name=modelName]").val(modelName);
				    $("#uploadFileForm_midVersion [name=oriVersion]").val(selectedVersionArr);
				    $("#uploadFileForm_midVersion [name=desc]").val(midDesc);
				    $('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
				    $("#winUploadPro").window("open");// 打开进度条窗口
				    intervalGetProgress = window.setInterval(function(){
				    	getUploadProgress();// 定时读取进度
						var dom = $("#progressUploadFile");
						if(dom.length == 0) clearInterval(intervalGetProgress);
				    }, 1000);
				    
				    uploadWithProgress({
				    	url: "${ctx}/cell/version/uploadMidVersionFile.action",
				    	form: document.querySelector("#uploadFileForm_midVersion"),
				    	success: function(data){
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
				
				if (midPageType == 'edit'){
					
					var params = { 
							midVersion : midVersion,
							oriVersion:selectedVersionArr, 
							desc:midDesc,
							modelName:modelName
					};
				    $.post("${ctx}/cell/version/updateMidVersionInfo.action", params, function(data) {
				        if (data["success"]) {
							showMsg("success_msg",'<%=rb.getString("ChengGong")%>')
							$('.editFileUpgradeDiv').animate({right:'-1000px'},400,function(){
								$('.editFileUpgradeDiv').html("");
								grid.datagrid("reload");
							});
				        }else {
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
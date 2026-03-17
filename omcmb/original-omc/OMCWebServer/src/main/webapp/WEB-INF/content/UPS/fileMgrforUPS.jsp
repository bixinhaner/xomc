<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
.panelDefault .upgradeTabsMainPage{
/* 	top:40px; */
	left:20px;
}
#importFileDiv{
	position:absolute;
	width:900px;
	/* height:92%; */
	right:-1000px;
	background:#fff;
	top:0px;
	bottom:0px;
	z-index:100;
	box-shadow:2px 3px 16px rgba(158,200,222,0.5);
	overflow:auto;
	/* border:1px solid #4AB3FF; */
}
.forbidden > span{
   	background: #B0CBDD !important;
}
.editFileUpgradeDiv,.viewFileUpgradeDiv{
	position:absolute;
	width:900px;
	/* height:94%; */
	right:-1000px;
	background:#fff;
	top:0px;
	bottom:0px;
	z-index:100;
	box-shadow:2px 3px 16px rgba(158,200,222,0.5);
	overflow:auto;
	/* border:1px solid #4AB3FF; */
}

.slideDownAllDiv{
	position:absolute;
	top: 0;
	left: 0;
	bottom: 0;
	width:100%;
	height:100%;
	background:#fff;
	z-index:100;
	border:none;
	display:none;
}
.tabsTitle{
	border:none;
}
.showFileOp div{
	padding-left:14px;
	border-bottom:none;
	width:auto;
}
.el-badge{
	position:relative;
}
.el-badge__content{
	position:absolute;
	top:6px;
	right:-4px;
	transform:translateY(-50%) translateX(100%);
	background-color:transparent;
	border-radius:10px;
	color:#fff;
	display:inline-block;
	font-size:10px;
	height:12px;
	line-height:11px;
	padding:0 6px;
	text-align:center;
	white-space:nowrap;
	cursor:default;
	border:1px solid transparent;
}
.el-icon-star-badge:before{
	color:#F3916C;
}
</style>
<div class='panelDefault' style='overflow:hidden'>
	<!-- 右上角导入按钮 -->
	<div id='upgradeImportDiv' class="circleIcon placeholder-bt CODE_UPS hidden" style="right: 120px;" placeholder="<%=rb.getString("DaoRuWenJian")%>">		
		<span class="el-icon el-icon-circle-import" onclick="importUpgradeFile()"></span>
	</div>
	<div id='upgradeAddDiv' class="circleIcon placeholder-bt CODE_UPS hidden" style="right: 60px;" placeholder="<%=rb.getString("XinJianRenWu")%>">		
		<span class="el-icon el-icon-circle-addTask" onclick="toAddUPS()"></span>
	</div>
	<div id='upgradeViewDiv' class="circleIcon placeholder-bt" placeholder="<%=rb.getString("RenWuLieBiao")%>">		
		<span class="el-icon el-icon-circle-taskList" onclick="toViewUPS()"></span>
	</div>

	<div id="upsSoftOperateDiv" class="slideDownAllDiv">
		<div id="ups_slide_content"></div>
	</div>
	<div class="singleTitle">
		<span>UPS&nbsp;<%=rb.getString("ShengJi")%>&nbsp;<%=rb.getString("WenJian")%></span>	
	</div>
	<div class="singleContentDiv">
		<table id='fileinfo_upgradeups'></table>
	</div>
	<!-- 导入文件模块 -->
	<div id='importFileDiv' class="slidebarPanel"></div>
	<!-- 新增文件模块 -->
	<div id="winAddUpgradTask" class='slidebarPanel' ></div>
	<!-- 修改文件模块 -->
	<div class='editFileUpgradeDiv slidebarPanel' ></div>
	<!-- 查看文件模块 -->
	<div class='viewFileUpgradeDiv slidebarPanel'></div>
</div>
<%-- 下载文件的用的表单 --%>
<form method="post" style="display: none"  id="downloadFileForm_filelib"></form>

<%-- 上传文件的用的表单 --%>
<form enctype="multipart/form-data" method="post" id="uploadFileForm_filelib">
    <input name="newFileName" id="newFileName" value="" hidden="true">
    <input name="fileSize" id="fileSize" value="" hidden="true">
    <input name="fileType" id="fileType" value="" type="hidden"/>
    <input name="deviceType" value="" type="hidden"/>
    <input name="product" value="" type="hidden"/>
    <input name="version" value="" type="hidden"/>
    <input name="to_who" value="" type="hidden"/>
    <input name="desc" value="" type="hidden"/>
    <input name="uploadFile" id="uploadFile_filelib"  type="file" style="display: none;">
    <input name="md5" value="" type="hidden"/>
     <input name="recommend" value="" type="hidden"/>
</form>
<!-- 工具栏--按查询  -->
<div id="toolbar_oducpeUpgrade" class="toolbarContainer" style="position:relative">
     <div class="queryGroup">
     	<input id='oducpeInput'  placeholder="<%=rb.getString("BanBen")%>">
		<b class="el-icon el-icon-common-search" onclick='$("#fileinfo_upgradeups").datagrid("reload")'></b>
     </div>
</div>

<!-- 菜单生成 -->
<div class="wrap">
    <div id="enbSoftwareFileMenu" class="showFileOp"></div>
</div>
<script>
	var isCpe = '${isUpsFileUpload}';
	var operateType = 'ups';
	function slideUPSDiv(bool,isHidden){
		if(bool == true) {
			var url = '${ctx}/cell/version/toUpsVersionMgr.action',
				$conDiv = $('#ups_slide_content');
			$conDiv.html('');
			$conDiv.load(url,function(data){
				$.parser.parse(this);
				if(isHidden == true) openWinAddTask();
			});
			if(isHidden != true) {
				$("#upsSoftOperateDiv").slideDown(500,function(){
					$conDiv.resize();
				});
			}
		}else {
			$("#upsSoftOperateDiv").slideUp(500);
		}
	}
	function toAddUPS(){
		slideUPSDiv(true,true);
	}
	function toViewUPS(){
		slideUPSDiv(true);
	}
	
	$(function(){
		closeLoading();
		setTimeout(function(){
			isJumpToPage = {};
		},1000)
		$("#fileinfo_upgradeups").datagrid({ // 列表生成
			url:'${ctx}/cell/version/queryfileInfosList.action?file_type=5',
			queryParams:{timeZone:timeZone},
			idField:'id',
			fit:true,
			fitColumns:true,
			nowrap:false,
			border:false,
			striped:true,
			singleSelect:true,
			rownumbers:true,
			pagination:true,
			pagePosition:'bottom',
			pageSize:100,
			pageList:[50,100,200,500],
			toolbar:'#toolbar_oducpeUpgrade',
			onBeforeLoad:function(param){
				param.searchText = $("#oducpeInput").val()
			},
			onLoadSuccess:function(){
				var code = isJumpToPage?isJumpToPage.code:'';
				var vid = isJumpToPage?isJumpToPage.vid:'';
				if(!code){
				}else if(code == 'cpe_odu'&& $(".upgradeLists span[logtype=upgradecpe]").is(":visible")){
					if(vid){
						var index = $("#fileinfo_upgradeups").datagrid("getRowIndex",vid);
						$("#fileinfo_upgradeups").datagrid("selectRow",index);
					}
				}
				$(this).datagrid("fixRownumber");
				$(this).datagrid("enableContextmenuAutoSize");
			},
			columns:[[
				{field:'id',hidden:true},
				{field:'operation',formatter:moreCpeFileFormatter,styler:setStyle,fixed:true, width:30,title:''},
				{field:'version',formatter:versionFmt, width:250,title:'<%=rb.getString("BanBen")%>'},
				{field:'product', width:150,title:'<%=rb.getString("ChanPinLeiXingBiaoZhi")%>'},
				{field:'size', width:100,title:'<%=rb.getString("WenJianDaXiao")%>'},
				{field:'upload_time', width:140,title:'<%=rb.getString("ShangChuanShiJian")%>'}
			        ]]
		})

		$("#oducpeInput").bind("keyup",function(e){
			if(e.keyCode == 13){
				$("#fileinfo_upgradeups").datagrid("reload");
			}
		})
		$(document).click(function(e){
		 	var e = e || window.event;
	        var elem = e.target || e.srcElement;
	        while(elem){
	            if($(elem).hasClass('el-icon-operation-more') || elem.className == 'showFileOp'|| elem.className == 'showFileUbootOp'|| elem.className == 'showFilePatchOp'|| elem.className == 'showCpeOduFileOp'|| elem.className == 'showCpeIduFileOp'){
	                return
	            }
	            elem = elem.parentNode;
	        }
	        //$(".showFileOp").css('display','none');
	        $(".showFileUbootOp").css('display','none');
	        $(".showFilePatchOp").css('display','none');
	        $(".showCpeOduFileOp").css('display','none');
	        $(".showCpeIduFileOp").css('display','none');
			$('#enbSoftwareFileMenu').hide();
		});
	})
	/**
	 * 版本转换
	 * @param value:默认值
	 * @param rowData:传入值
	*/
	function versionFmt(value,rowData,rowIndex){
		if(rowData.recommend == '1'){
			return "<div class='el-badge'><span>"+value+"</span><span class='el-badge__content el-icon el-icon-star-badge'></span></div>"
		}else{
			return value;
		}
	}
	/**
	 * 操作下拉详情
	 * @param rowId:当前传入的ID
	*/
	function enbSoftwareFileOp(rowId,fileName,type,recommend,el){
		var XiaZai = "<%=rb.getString("XiaZai")%>",
			XinXi = "<%=rb.getString("XinXi")%>",
			XiuGai = "<%=rb.getString("XiuGai")%>",
			ShanChu = "<%=rb.getString("ShanChu")%>";
		var recommendIcon = '';
		if(recommend == '1'){//说明此文件是推荐文件
			var TuiJian = '<%=rb.getString("QuXiaoTuiJian")%>';
			recommendStr = '0';
			recommendIcon = 'el-icon el-icon-operation-cancel-recommend CODE_UPS hidden';
		}else{
			var TuiJian = '<%=rb.getString("TuiJian")%>';
			recommendStr = '1';
			recommendIcon = 'el-icon el-icon-operation-recommend CODE_UPS hidden';
		}
		/* 菜单显示控制 */
		
		var data = [
				{rowId: rowId, type: type, fileName: fileName, text: XinXi, code: 'view', cls: 'el-icon el-icon-operation-info'},
				{rowId: rowId, type: type, fileName: fileName, text: XiaZai, code: 'download', cls: 'el-icon el-icon-operation-download'},
				{rowId: rowId, type: type, fileName: fileName, text: XiuGai, code: 'modify', cls: 'el-icon el-icon-operation-edit CODE_UPS hidden'},
				{rowId: rowId, type: type, fileName: fileName, text: ShanChu, code: 'remove', cls: 'el-icon el-icon-operation-delete CODE_UPS hidden'},
				{rowId: rowId, type: type, recommend: recommendStr, text: TuiJian, code: 'recommend', cls: recommendIcon}
			];
		$('#enbSoftwareFileMenu').cmenu({data: data, click: enbSoftwareFileOpClick}); 
		/* 菜单位置 */
		var allHeight = $(document).height(),
			isTabsShow = $('.omcPageTitleDiv:first').is(':visible'),
			tabsHeight = isTabsShow?$('.omcPageTitleDiv:first').height():0,
			thisTop = $(el).offset().top;
		if((allHeight - thisTop) <200){
			$('#enbSoftwareFileMenu').css({
				"top":thisTop - 179 - tabsHeight,
				"left":30,
			});
			if((allHeight - thisTop) <184) $('.item-child ').css({"top":"-54px",});
		}else{
			$('#enbSoftwareFileMenu').css({
				"top":thisTop - 20 - tabsHeight,
				"left":30,
			});
		}
		
		$('#enbSoftwareFileMenu').show();
	}
	/**
	 * 点击具体操作项
	 * @param row:点击的当前项
	*/
	function enbSoftwareFileOpClick(row){
		var codes = {
				view: viewFile,
				download: fileLibraryDownload,
				modify: editFile,
				remove: delFile,
				recommend : recommendFile
			},
			fileTypes = {
				fileinfo_upgradeups: 4
				
			},
			fileType = fileTypes[row.type],
			fileName = row.fileName;
		if(codes[row.code]) {
			if(row.code == 'download') codes[row.code](fileName,fileType);
			else if(row.code == 'recommend') codes[row.code](row.rowId,row.recommend,row.type)
			else codes[row.code](row.rowId,row.type);
		}
		$('#enbSoftwareFileMenu').hide();
	}
	/**
	 * 更多操作按钮
	 * @return value:生成DOM元素
	 * @param rowData:当前的表格数据
	*/
	function moreCpeFileFormatter(value,rowData,rowIndex){
		var row_id = rowData.id;
		var fileName = rowData.file_name;
		var recommend = rowData.recommend;
		if(value == null){
			value = "<div class='el-icon el-icon-operation-more' title='"+ CaoZuo+"' onclick='enbSoftwareFileOp("+ row_id +",\""+fileName+"\",\"fileinfo_upgradeups\",\""+recommend+"\",this)' ></div>";
		}
		return value;
	}
	// 没有用的地方
	function openMoreOperation(idVal,e,tableId,div,showOp){
		var thisTop = $(e).offset().top;
		var allHeight = $(document).height();
		var indexRow = $("#"+tableId).datagrid("getRowIndex",idVal);
		var rowHeight = $("."+div+" .datagrid-view2").find("tr[datagrid-row-index="+indexRow+"]").height();
		if((allHeight - thisTop) < 240){
			$(e).next("."+showOp).css("bottom",rowHeight+"px");                                                                                                                                                                                                                                                             
		}else{
			$(e).next().css("top",(rowHeight)+"px");
		}
		var current = $('.'+showOp,$(e).parent());
		$('.'+showOp).not(current).hide();
		current.fadeToggle(100);
	}
	//设置操作列单元格样式 
	function setStyle(){
		return 'position:relative';
	}
	function importUpgradeFile(){ // 导入按钮
		closeFileUpgradeWindow();
		closeViewFileUpradeWindow();
		/* $(".upgradeLists > span").each(function(index,ele){
	        if($(this).hasClass("active")){
	            choseType = $(this).attr("logtype");
	        }
	    }) */
		document.getElementById("uploadFile_filelib").value = "";// 置空文件组件
	    $("#uploadFileForm_filelib [name=fileType]").val("ups");
	    $("#winUpgradeFileInfo .file_info").val("");
		$("#importFileDiv").animate({right:"0px"},400,function(){
			$("#importFileDiv").panel({
				href:'${ctx}/cell/version/loadPage.action',
				width:900
			})
		});
	}
	/**
	 * 取消程序
	 * @param idVal:id
	 * @param recommend:当前行的data.recommend
	*/
	function recommendFile(idVal,recommend,gridId){
		$.post("${ctx}/cell/version/updateRecommendStatus.action", {id: idVal,recommend:recommend}, function(data) {
            if (data["success"]) {
                 $("#" + gridId).datagrid("reload");
            } else {
                $.messager.alert(TiShi, data["message"]);
            }
        }, "json");
	}
	/**
	 * 删除已上传的升级文件 
	 *  @param idVal:修改值
	 * @param gridId:修改的ID
	 * */ 
	function delFile(idVal, gridId) {
		closeWindow();
	    $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuWenJian")%>", function(r) {
	        if (r) {
	            $.post("${ctx}/cell/version/deleteVersionFile.action", {fileID: idVal}, function(data) {
	                if (data["success"]) {
	                     $("#" + gridId).datagrid("reload");
	                } else {
	                    showMsg('error_msg',data["message"]);
	                }
	            }, "json");
	        }
	    }).addClass("seriousConfirm");
	}
	/**
	 * 查看文件
	 * @param idVal:当前数据ID
	*/
	function viewFile(idVal){
		//$(".showFileOp").hide();
		closeFileUpgradeWindow();
		closeImportFileWindow();
		$(".viewFileUpgradeDiv").animate({right:'0px'},400,function(){
			$(".viewFileUpgradeDiv").panel({
				width:900,
				href:'${ctx}/cell/version/loadPage.action?code=view',
				onLoad:function(){
					$.post("${ctx}/cell/version/getDeviceVersionFileInfo.action", { versionId : idVal }, function(data) {
						if (data) {
							$("#winViewUpgradeFileInfo input[name='fileType']").val(data.file_type);
							$("#winViewUpgradeFileInfo #filePathView").val(data.file_name);
							$("#winViewUpgradeFileInfo #productView").val(data.product);
							$("#winViewUpgradeFileInfo #versionView").val(data.version);
							$("#toWhoView").combobox("setValue",data.toWho);
							$("#recommendView").combobox("setValue",data.recommend);
							$("#winViewUpgradeFileInfo #descView").val(data.desc);				
						}
					}, "json");
				}
			})
		});
	}
	/**
	 *  修改已上传的升级文件 
	 * @param idVal:修改值
	 * @param gridId:修改的ID
	 * */
	function editFile(idVal, gridId) {
		//加载设备软件版本文件信息
		//$(".showFileOp").hide();
		$(".showCpeOduFileOp").hide();
		$(".showCpeIduFileOp").hide();
		closeViewFileUpradeWindow();
		closeImportFileWindow();
		$(".editFileUpgradeDiv").animate({right:'0px'},400,function(){
			$(".editFileUpgradeDiv").panel({
				width:900,
				href:'${ctx}/cell/version/loadPage.action?code=edit',
				onLoad:function(){
					$.post("${ctx}/cell/version/getDeviceVersionFileInfo.action", { versionId : idVal }, function(data) {
						if (data) {
							$("#winEditUpgradeFileInfo input[name='fileType']").val(data.file_type);
							$("#winEditUpgradeFileInfo #filePathEdit").val(data.file_name);
							$("#winEditUpgradeFileInfo #productEdit").val(data.product);
							$("#winEditUpgradeFileInfo #versionEdit").val(data.version);
							$("#recommendEdit").combobox("setValue",data.recommend);
							$("#toWhoEdit").combobox("setValue",data.toWho);
							$("#winEditUpgradeFileInfo #descEdit").val(data.desc);				
						}
					}, "json");
					$("#submitUploadEdit").unbind("click");
					$("#submitUploadEdit").click(function(){
						var required_err = false;
						var productEditDom = $("#winEditUpgradeFileInfo #productEdit"),
							productEditValue = productEditDom.val(),
							deviceEditDom = $("#winEditUpgradeFileInfo #deviceEdit"),
							deviceEditValue = '';
						$("#winEditUpgradeFileInfo #productEdit").val(productEditValue);
							if(checkVersion()){
								if($("#productEdit").val() == ""){
									$("#selectType").css("visibility","visible");
									return false;
								}else{
									$("#selectType").css("visibility","hidden");
								}
								
								//修改设备软件版本文件信息
								var pram={ 
										versionId : idVal,
										fileType:$("#winEditUpgradeFileInfo input[name='fileType']").val(),
										fileName:$("#winEditUpgradeFileInfo #filePathEdit").val(),
										product:productEditValue,
										deviceType:deviceEditValue,
										version:$("#winEditUpgradeFileInfo #versionEdit").val(), 
										toWho:$("#toWhoEdit").combobox("getValue"),
										recommend:$("#recommendEdit").combobox("getValue"),
										desc:$("#winEditUpgradeFileInfo #descEdit").val()
								};
							    $.post("${ctx}/cell/version/goModifyDeviceVersionFileInfo.action", pram, function(data) {
							        if (data["success"]) {
										showMsg('success_msg','<%=rb.getString("ChengGong")%>');
										$('.editFileUpgradeDiv').animate({right:'-1000px'},400,function(){
											$('.editFileUpgradeDiv').html("");
											$("#" + gridId).datagrid("reload");
										});
							        }
							    }, "json");
							}
					});
				}
			})
		});
	}
	/**
	 * 下载文件
	 * @param fileName:当前下载数据名称
	 * @param fileType:当前下载数据类型
	*/
	function fileLibraryDownload(fileName, fileType){
	    $.post("${ctx}/omc/version/file/fileIsExist.action", { fileName : fileName, fileType : 5}, function (data) {
	        if(data["success"]){
	        	/* $("#downloadFileForm_filelib").form('submit', {
	                url: "${ctx}/omc/version/file/downLoadFile.action",
	                onSubmit: function(param){
	                    param.fileName = fileName;
	                    param.fileType = 5;
	                }
	            }); */
	            exportByForm("${ctx}/omc/version/file/downLoadFile.action",{
	            	fileName: fileName,
	            	fileType: 5
	            });
	        }else{
	        	showMsg('error_msg',data["message"]);
	        }
	    },"json");
	}
	// 关闭修改文件模块 	
	function closeFileUpgradeWindow(){
		$(".editFileUpgradeDiv").animate({right:'-1000px'},400,function(){
			$('.editFileUpgradeDiv').html("");
		});
	}
	// 关闭查看文件模块
	function closeViewFileUpradeWindow(){
		$(".viewFileUpgradeDiv").animate({right:'-1000px'},400,function(){
			$('.viewFileUpgradeDiv').html("");
		});
	}
	//关闭导入文件模块
	function closeImportFileWindow(){
		$("#importFileDiv").animate({right:"-1000px"},400,function(){
			$("#importFileDiv").html("");
		});
	}
	// 关闭所有模块
	function closeWindow(){
		closeFileUpgradeWindow();
		closeViewFileUpradeWindow();
		closeImportFileWindow();
	}
	
	/**
	 * 任务执行结果格式化
	 * @param value：传入值
	*/
	function taskResultFmt(value, rowData, rowIndex) {
		if (value == "0") {
			return "<%=rb.getString("ChengGong")%>";
		} else if (value == "1") {
			return "<%=rb.getString("BuFenChengGong")%>";
		} else if (value == "2") {
			return "<%=rb.getString("ShiBai")%>";
		} else if (value == "3") {
			return "<%=rb.getString("ZhongZhi")%>";
		} else {
			return "";
		}
	}

	/**
	 * 任务进度格式化
	 * @param value：传入值
	*/
	function taskProgressFmt(value, rowData, rowIndex) {
		if (value == "0") {
			return "<%=rb.getString("WeiKaiShi")%>";
		} else if (value == "1") {
			return "<%=rb.getString("JinXingZhong")%>";
		} else if (value == "2") {
			return "<%=rb.getString("YiJieShu")%>";
		}
	}
	
	// 显示md5值
	function showMd5Val(okHandler, cancelHandler, md5) {
	    var url = '${ctx}/cell/version/loadPage.action?code=md5';;
		openDefaultWindow(url,{
			title: '<%=rb.getString("WenJianXinXi")%>',
			width:410,height:200,
			onLoad: function(){
				$("#cancelBtn_md5").unbind("click");
				$("#okBtn_md5").unbind("click");
				$("#cancelBtn_md5").bind("click", cancelHandler);
				$("#okBtn_md5").bind("click", okHandler);
			    $("#MD5Value").html(md5);
			}
		}); 
	}
	// 客户校验MD5值之后，确认上传文档
	function saveInputFile() {
		var params = {};
		params = {
			fileName_temp : retMap["fileName_temp"],
			fileSize : retMap["fileSize"],
			fileType : retMap["fileType"],
			product : retMap["product"],
			version : retMap["version"],
			desc : retMap["desc"],
			MD5 : retMap["MD5"],
			destination : retMap["destination"]
		};

		var grid;

		var fileType = $("#uploaFileForm_filelib [name=fileType]").val();

		if(fileType == "upgradecpe") {
			grid = $("#fileinfo_upgradeups");
		}

		$.post("${ctx}/cell/version/saveInputFileForSure.action", params,
				function(data) {
					if (data["success"]) {
						/* $("#inputFileMd5").window("close"); */
						closeDefaultWindow();
						showMsg('success_msg',data["message"]);
						grid.datagrid("reload");
					}
				}, "json");
	}

	// 客户校验MD5值之后，认为不需要上传该文件
	function cancelInputFile() {
		var param = {};
		param = {
			destination : retMap["destination"]
		};
		$.post("${ctx}/cell/version/cancelInputFileForSure.action", param,
				function(data) {
					if (data["success"]) {
						/* $("#inputFileMd5").window("close"); */
						closeDefaultWindow();
					}
				}, "json");
	}

</script>
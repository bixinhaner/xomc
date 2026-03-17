<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style type="text/css">
  .borderBoxClass {
    box-sizing: border-box;
    -moz-box-sizing: border-box;
    -webkit-box-sizing: border-box;
  }
</style>
<div class="easyui-layout" data-options="fit:true">
  <div region="west" style="padding-right: 15px;background-color: #F3F3F4;" data-options="border:false,width:200">
	  <div class="easyui-panel" data-options="border:true,fit:true">
	  		<div data-options="region:'north',border:false,height: 40,collapsible:false" >
		        <div class="omcPageTitleDiv" style="padding:0 10px 0 0px">
					<ul class="omcPageTitleContainer">
						<li class="default"><%=rb.getString("CeLiangDingZhi")%></li>
					</ul>
				</div>
		</div>
		   <div class="easyui-layout" data-options="fit:true,border:false">
		     <div region="north" style="border-width:0px 0px 1px 0px;padding: 20px 0px;" data-options="border:true,collapsible:false,height:80">
			  	<a onclick="goCustomizeMRWin()" class="linkbutton" style="float:right;margin-right:10px;"><span><%=rb.getString("DingZhi")%></span></a>
		     </div>
		     <div region="center" data-options="border:false" style="padding-top: 10px;">
		       <ul id="mrFileTypeTree" data-options="border:false"></ul>
		     </div>
		   </div>
	   </div>
  </div>
  <div region="center" data-options="border:false,collapsible:false">
  		<div data-options="region:'north',border:false,height: 40,collapsible:false" >
		        <div class="omcPageTitleDiv">
					<ul class="omcPageTitleContainer">
						<li class="default"><%=rb.getString("WenJianLieBiao")%></li>
					</ul>
				</div>
		</div>
	   <div class="easyui-layout" data-options="border:false,fit:true">
	     <div region="north" data-options="border:false,height:80" style="padding:20px">
	         <div class="linkbuttonGroup" style="float:right;">
	         	<a onclick="downloadFiles()" class="linkbutton"><span><%=rb.getString("XiaZai")%></span></a>
	         	<a onclick="clearFiles()" class="linkbutton"><span><%=rb.getString("QingChuWenJian")%></span></a>
	         </div>  
	     </div>
	     <div region="center" data-options="border:false" style="padding: 0 15px;">
	       <table id="mrFilesDatagrid" class="easyui-datagrid"
	              data-options="border:false,
	                            fit:true,
	                            checkbox:true,
	                            striped: true,
	                            pagination: true,
	                            pagePosition: 'bottom',
	                            rownumbers: true,
	                            fitColumns: true,onLoadSuccess:datagridLoadSuccess">
	         <thead>
	          <tr>
	            <th data-options="field:'ck',checkbox:true"></th>
	            <th data-options="field:'fileName'" width="100"><%=rb.getString("WenJianMing")%></th>
	            <th data-options="field:'modifyTime'" width="100"><%=rb.getString("XiuGaiShiJian")%></th>
	          </tr>
	         </thead>
	       </table>
	     </div>
	   </div>
  </div>
</div>
<%-- 窗口，定制小站是否上报MR文件--%>
<div id="winCustomizeMR" class="easyui-window" title="<%=rb.getString("DingZhiMR")%>"
	data-options="modal:true,closed:true,collapsible:false,minimizable:false, maximizable:false,width:860,height:500">
	<div class="easyui-layout" data-options="border:false,fit:true">
		<div region="north" style="height: 100px;padding: 10px;">
			<div style="padding: 5px 10px 5px 20px; width: 300px; display: inline-block;">
				<label style="width: 120px; display: inline-block" class="borderBoxClass"><%=rb.getString("TongJiZhouQi")%><%=rb.getString("MaoHao")%></label>
				<select id="customizeMRStaticPeriod" class="easyui-combobox border border-box borderBoxClass"
					style="height: 26px" data-options="editable:false">
					<option value="5" selected>5</option>
					<option value="15">15</option>
					<option value="30">30</option>
					<option value="60">60</option>
				</select>
			</div>
			<div style="padding: 5px 10px 5px 20px; width: 300px; display: inline-block;">
				<label style="width: 70px; display: inline-block" class="borderBoxClass"><%=rb.getString("MRLeiXing")%><%=rb.getString("MaoHao")%></label>
				<input type="checkbox" name="customizeMRType" class="borderBoxClass" style="margin-left: 5px;" value="MRS" />MRS
				<input type="checkbox" name="customizeMRType" class="borderBoxClass" style="margin-left: 5px;" value="MRO" />MRO
				<input type="checkbox" name="customizeMRType" class="borderBoxClass" style="margin-left: 5px;" value="MRE" />MRE
			</div>
			<div style="padding: 5px 10px 5px 20px; width: 300px; display: inline-block;">
				<label style="width: 120px; display: inline-block" class="borderBoxClass"><%=rb.getString("CaiYangZhouQi")%><%=rb.getString("MaoHao")%></label>
				<select id="customizeMRSamplePeriod" class="easyui-combobox border border-box borderBoxClass"
					style="height: 26px" data-options="editable:false">
					<option value="2048" selected>2048</option>
					<option value="5120">5120</option>
					<option value="10240">10240</option>
					<option value="60000">60000</option>
					<option value="360000">360000</option>
					<option value="720000">720000</option>
					<option value="1800000">1800000</option>
					<option value="3600000">3600000</option>
				</select>
			</div>
			<div style="padding: 5px 10px 5px 20px; width: 300px; display: inline-block;">
				<label style="width: 70px; display: inline-block" class="borderBoxClass"><%=rb.getString("CaoZuoLeiXing")%><%=rb.getString("MaoHao")%></label>
				<select id="operation_type" class="easyui-combobox border border-box borderBoxClass" style="height: 26px" data-options="editable:false">
					<option value="0" selected><%=rb.getString("GuanBi")%></option>
					<option value="1"><%=rb.getString("KaiQi")%></option>
					<option value="2"><%=rb.getString("QuanBu")%><%=rb.getString("GuanBi")%></option>
					<option value="3"><%=rb.getString("QuanBu")%><%=rb.getString("KaiQi")%></option>
				</select>
				<input id="exits_customize" type="hidden" value="0"/>
			</div>
			<a onclick="confirmCustomizeMR()" class="linkbutton linkbutton_trend" style="margin-right: 15px;float:right;"><span><%=rb.getString("DingZhi")%></span></a>
		</div>
		<div region="center" data-options="border:false" style="padding: 10px 10px 5px 10px">
			<div class="queryGroup" style="padding:0 0 20px 20px;">
				<input id="search_text" type="text" placeholder="<%=rb.getString("XiaoZhanBianMa")%>"/>
				<b onclick="$('#currentCustomizeMrReportTaskDatagrid').datagrid('reload')"></b>
			</div>
			<div style="height: 87%;" data-options="border:false">
				<table id="currentCustomizeMrReportTaskDatagrid"></table>
			</div>
		</div>
	</div>
</div>

<%-- 表单-下载mr文件 --%>
<form id="formDownloadMrFiles" style="display:none" method="post" action="${ctx}/cell/perfmgmt/mr/downloadMRFiles.action">
  <%-- //已选中的文件路径，提交表单之前为此值赋值 --%>
  <input id="mrFilesTimeZone" name="mrFilesTimeZone" type="hidden" value="">
  <input id="mrFilesPath" name="mrFilesPath" type="hidden" value="">
  <input id="mrFilesName" name="mrFilesName" type="hidden" value="">
</form>
<script type="text/javascript">
	//保存当前已选择的基站对象数组
	var delCellCodeObjArr = new Array();
	$(function() {
		closeLoading();
		
		var taskTypeTreeData = [ {
			"text" : "MRS",
			"url" : "/cell/perfmgmt/mr/getMRFilesTreeNodes.action?MRType=MRS"
		}, {
			"text" : "MRO",
			"url" : "/cell/perfmgmt/mr/getMRFilesTreeNodes.action?MRType=MRO"
		}, {
			"text" : "MRE",
			"url" : "/cell/perfmgmt/mr/getMRFilesTreeNodes.action?MRType=MRE"
		} ];
		
		createSecondMenu(taskTypeTreeData, typeTreeOnClick, $("#mrFileTypeTree"));
		
		$("#mrFilesDatagrid").datagrid({
			url: '${ctx}/cell/perfmgmt/mr/getMRFilesTreeNodes.action?MRType=MRS',
            queryParams:{timeZone:timeZone}
        });
	});
	
   //MR类型树，节点点击事件
   function typeTreeOnClick(node) {
    	$("#mrFilesDatagrid").datagrid({
    		url: "${ctx}" + node.attr("url"),
    		pageNumber : 1
    	});
   }
   
  <%--点击Clear按钮，清除选中文件--%>
  function clearFiles() {
    var mrFilesChecked = $("#mrFilesDatagrid").datagrid("getSelections");
    if ((0 == mrFilesChecked.length || null == mrFilesChecked)) {
      showMsg('prompt_msg',"<%=rb.getString("QingXianXuanZeWenJian")%>");
      return;
    }
    var filePathStr = "";
    var fileNameStr = "";
    for (var mrFileCount = 0; mrFileCount < mrFilesChecked.length; mrFileCount++) {
      filePathStr += mrFilesChecked[mrFileCount].id + ",";
      fileNameStr += mrFilesChecked[mrFileCount].fileName + ",";
    }
    filePathStr = filePathStr.substring(0, filePathStr.length - 1);
    fileNameStr = fileNameStr.substring(0, fileNameStr.length - 1);

    $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("QueDingShanChuWenJian")%>", function (r) {
      if (r) {
        var params = {
          "filePathStr": filePathStr,
          "fileNameStr": fileNameStr
        }
        $.post("${ctx}/cell/perfmgmt/mr/doClearMRFiles.action", params, function (data) {
          if (data["success"]) {
            $("#mrFilesDatagrid").datagrid("clearSelections");
            $("#mrFilesDatagrid").datagrid("reload");
          }
        }, "json");
      }
    }).addClass("seriousConfirm");
  }
  <%--点击Download按钮，下载文件到本地--%>
  function downloadFiles() {
    var mrFilesChecked = $("#mrFilesDatagrid").datagrid("getChecked");
    if ((0 == mrFilesChecked.length || null == mrFilesChecked)) {
      showMsg('prompt_msg',"<%=rb.getString("QingXianXuanZeWenJian")%>");
      return;
    }

    var filePathStr = "";
    var fileNameStr = "";
    for (var mrFileCount = 0; mrFileCount < mrFilesChecked.length; mrFileCount++) {
      filePathStr += mrFilesChecked[mrFileCount].id + ",";
      fileNameStr += mrFilesChecked[mrFileCount].fileName + ",";
    }
    filePathStr = filePathStr.substring(0, filePathStr.length - 1);
    fileNameStr = fileNameStr.substring(0, fileNameStr.length - 1);
    $("#mrFilesTimeZone").val(timeZone);
    $("#mrFilesPath").val(filePathStr);
    $("#mrFilesName").val(fileNameStr);
    /* $("#formDownloadMrFiles").form('submit',{
    	onSubmit: function(param){
			var bool = checkParams(param)
			if(!bool) return false;
		}
    }); */
    exportByForm($("#formDownloadMrFiles").attr("action"),{
    	mrFilesTimeZone:　timeZone,
    	mrFilesPath: filePathStr
    	mrFilesName: fileNameStr
    });
  }
  
  <%--弹出定制小站上报MR的页面--%>
  function goCustomizeMRWin(){
    $("#winCustomizeMR").window({
    	height: $(document).height() * 0.7
    }).window("center").window("open");
   
    layoutOnComplete();
   
    //刷新当前运营商的MR定制情况
    refreshCurrentCustomizeMRReport();
    
    $("#currentCustomizeMrReportTaskDatagrid").datagrid({
	    	border : false,
	    	fit : true,
	    	url : '${ctx}/cell/perfmgmt/mr/getCellCustomizeMRReportPage.action',
			queryParams : {
				timeZone : timeZone
			},
			checkbox : true,
			rownumbers : true,
			pagination : true,
			pageSize: 15,
            pageList: [15,20,50,100],
			pagePosition : 'bottom',
			fitColumns : true,
			columns:[[
				  {field: 'ck', checkbox: true},
		          {field:'small_cell_code',hidden:true},
		          {field:'connection_status', sortable: true, fixed: true, formatter: function(value, rowData, rowIndex){
		        	  	if ("0" == value) {
		        			return "<div class='conn_off'><div>";
		        		} else if ("1" == value) {
		        			return "<div class='conn_on'><div>";
		        		} else {
		        			return value;
		        		}
		          }},
		          {field:'serial_number',width : 100,title : '<%=rb.getString("XiaoZhanBianMa")%>'},
		          {field:'host_name', width: 100, title: '<%=rb.getString("HostName")%>'},
		          
		          /* {field:'mr_type',width : 100,title : '<%=rb.getString("MRLeiXing")%>'},
		          {field:'static_period',width : 100,title : '<%=rb.getString("TongJiZhouQi")%>'},
		          {field:'sample_period',width : 100,title : '<%=rb.getString("CaiYangZhouQi")%>'},
		          {field:'operation_time',width : 110,title : '<%=rb.getString("CaoZuoKaiShiShiJian")%>'},
		          {field:'operation_name',width : 100,title : '<%=rb.getString("CaoZuoMingCheng")%>',
				           formatter:function (value, rowData, rowIndex) {
							    		  if("0" == value){
							    			  return "<%=rb.getString("QuXiaoDingZhi")%>";
							    		  } else {
							    			  $("#currentCustomizeMrReportTaskDatagrid").datagrid("selectRow",rowIndex);
							    			  return "<%=rb.getString("DingZhi")%>";
							    		  } 
						    	    }
		          },*/
		          
		          {field:'status', sortable: true, width : 100,title : '<%=rb.getString("DingZhi")%><%=rb.getString("ZhuangTai")%>',
			           formatter:function (value, rowData, rowIndex) {
			        	   			  // 0定制关闭 1定制开启 2正在开启定制 3开启定制失败 4正在关闭定制  5关闭定制失败 6定制下发超时
						    		  if("0" == value){
						    			  return "<%=rb.getString("DingZhi")%><%=rb.getString("GuanBi")%>";
						    		  } else if("1" == value) {
						    			  return "<%=rb.getString("DingZhi")%><%=rb.getString("ChengGong")%>";
						    		  } else if("2" == value) {
						    			  return "<%=rb.getString("KaiQi")%><%=rb.getString("DingZhi")%><%=rb.getString("JinXingZhong")%>";
						    		  } else if("3" == value) {
						    			  return "<%=rb.getString("KaiQi")%><%=rb.getString("DingZhi")%><%=rb.getString("ShiBai")%>";
						    		  } else if("4" == value) {
						    			  return "<%=rb.getString("GuanBi")%><%=rb.getString("DingZhi")%><%=rb.getString("JinXingZhong")%>";
						    		  } else if("5" == value) {
						    			  return "<%=rb.getString("GuanBi")%><%=rb.getString("DingZhi")%><%=rb.getString("ShiBai")%>";
						    		  } else if("6" == value) {
						    			  return "<%=rb.getString("DingZhi")%><%=rb.getString("ShiBai")%>";
						    		  } else{
						    			  return "<%=rb.getString("DingZhi")%><%=rb.getString("BuCunZai")%>";
						    		  } 
						    		
					    	    }
	          		}
		       ]],
		   onBeforeLoad : function(param){
			   var search_text = $("#search_text").val().trim();
		    	if(search_text != null && search_text != ""){
		    		param["search_text"] = search_text;
		    	}
		   },
		   onLoadSuccess:function(data){
			   $(this).datagrid("fixRownumber");
			   $(this).datagrid("enableContextmenuAutoSize");
		   }
		})
  }
  
  //刷新当前运营商的MR定制情况
  function refreshCurrentCustomizeMRReport(){
	  	//重置
	    resetCustomizeMRReportEvent();
	    
	    $.post("${ctx}/cell/perfmgmt/mr/getCurrentCustomizeMRReport.action", {timeZone:timeZone}, function (data) {
	        if (data) {
	        	$("#exits_customize").val("1");
	            $("#customizeMRStaticPeriod").combobox("setValue",data.static_period);
	            $("#customizeMRSamplePeriod").combobox("setValue",data.sample_period);
	            $("#operation_type").combobox("setValue",data.operation_name);
	            
	            var mr_type=data.mr_type;
	          	//遍历多选框,设置为不选状态
	            $("input[name='customizeMRType']").each(function () {
	            	var val=$(this).val();
	            	if(mr_type.indexOf(val)>=0){
	            		$(this).prop('checked', true);
	            	}
	            });
	        }else{
	        	$("#exits_customize").val("0");
	        }
	      }, "json");
  }
  
  /* 定制小站上报MR */
  function confirmCustomizeMR(){
	var exits_customize=$("#exits_customize").val();
	var operation_type=$("#operation_type").combobox("getValue");
	
	if ((operation_type=="0"||operation_type=="2") && exits_customize=="0"){
	    showMsg('prompt_msg',"<%=rb.getString("YunYingShang")%><%=rb.getString("BuCunZai")%><%=rb.getString("DingZhiMR")%>");
		return;
	}
	
	var mrParams = {
    		timeZone:timeZone,
	        operation_type:operation_type
    };
	if(operation_type=="1" || operation_type=="3" ){
		//统计周期
	    var staticPeriod = $("#customizeMRStaticPeriod").combobox("getValue");
	    //采样周期
	    var samplePeriod = $("#customizeMRSamplePeriod").combobox("getValue");
	    //mr类型
	    var mr_type = "";
	    //遍历多选框
	    $("input[name='customizeMRType']").each(function () {
		      if (true == this.checked) {
		    	  mr_type += this.value + ",";
		      }
	    });
	    if (mr_type == "") {
		    showMsg('prompt_msg',"<%=rb.getString("QingXuanZeMRLeiXing")%>");
		    return;
	    }
	    mr_type = mr_type.substring(0, mr_type.length - 1);
	    
	    mrParams["static_period"]=staticPeriod;
	    mrParams["sample_period"]=samplePeriod;
	    mrParams["mr_type"]=mr_type;
	   
	} 
	if (operation_type=="0" || operation_type=="1"){
	    var selectedCellArray = $("#currentCustomizeMrReportTaskDatagrid").datagrid("getSelections");
		if (selectedCellArray.length <= 0) {
			showMsg('prompt_msg',"<%=rb.getString("QingXuanZeSheBei")%>");
			return;
		}
		//拼接cellcode，serialNumber按“，”分隔
		var cellsStr = "";
		var serialNumberStr = "";
		var hostNameStr = "";
		$.each(selectedCellArray,function(index,obj){
			cellsStr += obj.small_cell_code + ","
			serialNumberStr += obj.serial_number + ",";
			hostNameStr += obj.host_name + ",";
	 	})
	    cellsStr = cellsStr.substring(0, cellsStr.length - 1);
	    serialNumberStr = serialNumberStr.substring(0, serialNumberStr.length - 1);
	    hostNameStr = hostNameStr.substring(0, hostNameStr.length - 1);
	   
	    mrParams["smallCellStr"]=cellsStr;
	    mrParams["serialNumberStr"]=serialNumberStr;
	    mrParams["hostNameStr"]=hostNameStr;
	}
    
    $.messager.confirm("<%=rb.getString("QueRen")%>", "<%=rb.getString("DingZhiMRCaoZuoTiShi")%>", function (r) {
			      if (r) {
			        $.post("${ctx}/cell/perfmgmt/mr/goCustomizeMRReport.action", mrParams, function (data) {
			          if (data["success"]) {
			        	refreshCurrentCustomizeMRReport();
			        	$("#currentCustomizeMrReportTaskDatagrid").datagrid("reload");
			            //$("#winCustomizeMR").window("close");
			          }
			          showMsg('error_msg',data.message);
			        }, "json");
			      }
    	}).addClass("seriousConfirm");
  }
  
	//重置定制MR上报规则
	function resetCustomizeMRReportEvent() {
		//遍历多选框,设置为不选状态
		$("input[name='customizeMRType']").each(function() {
			$(this).prop('checked', false);
		});
		$("#customizeMRStaticPeriod").combobox("setValue", "60");
		$("#customizeMRSamplePeriod").combobox("setValue", "5120");
	}
</script>


<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<%@ include file="/common/loading.jsp" %>
<style>
.addTaskErrorTip{
	color:red;
	display:none;
}
</style>
<div class="slidebarTitleDiv" style="min-width: 900px;position: relative;">
	<div id='mmlScriptTaskListInfo' style="display: inline-block;"></div>
	<div class="circleIcon placeholder-bt" placeholder="<%=rb.getString("GuanBi")%>" onclick="cancelCreateMMLScriptTask()">
		<span class="el-icon el-icon-circle-close"></span>
	</div>
</div>
<div class="slideBody" style="min-width: 900px;">
	<!-- 新建MML脚本任务 -->
	<div id="addMMLScriptTaskSteps" class="easyui-panel" data-options="border:false,fit:true">
		<div id="step_1" class="easyui-layout" data-options="border:false,fit:true">
			<div class='info_mmlScripts'>
				<div data-options="border:false,height:66" style="padding:20px;">
					<!-- 任务名称 -->
					<label style="display: inline-block;width:72px;" class="borderBoxClass"><%=rb.getString("RenWuMingCheng")%></label>
					<input id="taskName_info" disabled type="text" class="border border-box"style="width:463px;margin-left:6px;" placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
				</div>
				<div>
					<div class="group-title not-extend" style='margin-top:20px'>
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("XuanZeZhiXingFangShi")%></span>
					</div>
					<div style='margin:15px 0px 0px 20px;'>
						<div id="executionMethods_info" class="verM">
							<div class="verM">
								<input type="radio" name="period" value="active" /><label style="width:295px;display:inline-block;"> <%=rb.getString("LiJiZhiXing")%></label>

								<input type="radio" name="period" value="suspend" /><label> <%=rb.getString("GuaQi")%></label>
							</div>
							<div class="dateRe" style="margin-top: 20px;position:relative">
								<input type="radio" name="period" value="timing" /><label style="margin-right:16px;"> <%=rb.getString("DingShiZhiXing")%></label>
								<input id="time_info" disabled type="text" class="border border-box" style="height: 26px;width:200px;">
							</div>
							<div class="verM" style="margin-top: 20px;position:relative">
								<input type="radio" name="period" value="period" /><label style="margin-right:16px;"> <%=rb.getString("ZhouQiRenWu")%></label>
								<input id="period_start_info" disabled type="text" class="border border-box" style="height: 26px;width:90px;">
								--
								<input id="period_end_info" disabled type="text" class="border border-box" style="height: 26px;width:90px;">
								：
								<input id="period_time_info" disabled type="text" class="border border-box" style="height: 26px;width:80px;">
							</div>
						</div>
					</div>
				</div>
				<div>
					<div class="group-title not-extend" style='margin-top:20px'>
						<span class="title-icon"></span>
						<span class="title-text"><%=rb.getString("ZhiXingCeLue")%></span>
					</div>
					<div style='margin:15px 0px 0px 20px;'>
						<div>
							<%=rb.getString("LiXianSheBei")%>
							<input id="offlineRetryEnable_info" disabled name="offlineRetryEnable_info"  value="true" class="easyui-checkbox border-box border" type="checkbox" style="width:14px;vertical-align:middle;"/>
							<%=rb.getString("DengDaiSheBeiShangXianChongShi")%>
							<input id="offlineRetryWaitTime_info" disabled type="text" class="border border-box"style="width:50px;margin-left:6px;">
							<%=rb.getString("FenZhongS")%>
						</div>
						<div style="margin:20px 0;">
							<%=rb.getString("ZaiXianSheBei")%>
							<input id="failedRetryEnable_info" disabled name="failedRetryEnable_info" class="border-box border" type="checkbox" style="width:14px;vertical-align:middle;"/>
							<%=rb.getString("PeiZhiShiBaiChongShi")%>

							<input id="failedRetryCount_info" disabled type="text" class=" border-box border" style="height:26px;width:50px;" />
							<%=rb.getString("JianGeCiShuChongShi")%>

							<input id="failedRetryWaitTime_info" disabled type="text" class="border-box border" style="height:26px;width:50px;" />
							<%=rb.getString("FenZhongS")%>
						</div>
					</div>
				</div>
			</div>
			<div>
				<div class='add_mmlScripts'>
					<div region="north" data-options="border:false,height:66" style="padding:20px;">
						<!-- 任务名称 -->
						<label style="display: inline-block;width:72px;" class="borderBoxClass"><%=rb.getString("RenWuMingCheng")%></label>
						<input id="taskName_addMMLScriptTask" type="text" class="border border-box" maxlength=50
								style="width:463px;margin-left:6px;" placeholder="<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>">
					</div>
					<div region="center" data-options="border:false" style="padding:0px 20px 10px;">
						<div class="group-title not-extend">
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("XuanZeJiaoBen")%></span>
						</div>
						<div style='margin:15px 0px 0px 20px;height:40px;'>
							<input id="MMLScriptFileInput" type="text" class="border border-box file_info" readonly="readonly" style="vertical-align:middle;padding-right:27px;width:350px;"/>
							<a class="el-icon el-icon-operation-import" style='vertical-align:middle;display:inline-block;margin:0 2px 0 -29px;' title="<%=rb.getString("XuanZeWenJian")%>" href="javascript: void(0)" onclick="scanClick_selectMMLScript()"
							style="vertical-align:middle; margin:0 2px 0 -29px;display:inline-block;width:23px;height:24px;background-color:#fff;">
							</a>

							<div id="MMLScriptFileInput_err" class="addTaskErrorTip"><%=rb.getString("ZhiZhiChiTxtWenJian")%></div>
						</div>
						<div class="group-title not-extend" style='margin-top:20px'>
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("XuanZeZhiXingFangShi")%></span>
						</div>
						<div style='margin:15px 0px 0px 20px;'>
							<div class="verM">
								<input type="radio" id="active_addMMLScriptTask" status="active" checked="true" name="taskStatus" onchange="setExeTimerEnable_addMMLScriptTask()"/>
								<label for="active_addMMLScriptTask" style="width:295px;display:inline-block;"><%=rb.getString("LiJiZhiXing")%></label>
								<input type="radio" id="suspend_addMMLScriptTask" status="suspend"  name="taskStatus" onchange="setExeTimerEnable_addMMLScriptTask()"/>
								<label for="suspend_addMMLScriptTask"><%=rb.getString("GuaQi")%></label>
							</div>
							<div  class="dateRe" style="margin-top: 20px;position:relative">
								<input type="radio" id="timing_addMMLScriptTask" status="timing" name="taskStatus" onchange="setExeTimerEnable_addMMLScriptTask()"/>
								<label for="timing_addMMLScriptTask" style="margin-right:16px;"><%=rb.getString("DingShiZhiXing")%></label>
								<input id="exeTime_addMMLScriptTask" class="easyui-datetimebox border-box border" style="height:26px;"
									data-options="disabled:true,editable:false,onSelect:selectTimeMML">
								<p class='timeError' style='color:red;font-size:12px;margin-left:120px;position:absolute;bottom:-18px;display:none'><%=rb.getString("QingXuanZeShiJian")%></p>
							</div>
							<div class="verM" style="margin-top: 20px;position:relative">
								<input type="radio" id="period_addMMLScriptTask" status="period" name="taskStatus" onchange="setExeTimerEnable_addMMLScriptTask()"/>
								<label for="period_addMMLScriptTask" style="margin-right:16px;display: inline-block;width: 85px;"><%=rb.getString("ZhouQiRenWu")%></label>
								<input id="periodStartTime_addMMLScriptTask" class="easyui-datebox border-box border" style="height:26px;width: 110px;"
									data-options="editable:false,onSelect:selectStartTimeMML">
								--
								<input id="periodEndTime_addMMLScriptTask" class="easyui-datebox border-box border" style="height:26px;width: 110px;"
									data-options="editable:false,onSelect:selectEndTimeMML">
								：
								<input id="periodTime_addMMLScriptTask" class="easyui-timespinner border-box border" style="height:26px;width: 100px;"
									data-options="showSeconds:true,onChange:selectPeriodTimeMML">

								<p class='periodTimeError' style='color:red;font-size:12px;margin-left:125px;position:absolute;bottom:-18px;display:none'>
									<%=rb.getString("ZhouQiShiJianTiShi")%>
								</p>
							</div>
						</div>

						<div class="group-title not-extend" style='margin-top:20px'>
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("ZhiXingCeLue")%></span>
						</div>
						<div style='margin:15px 0px 0px 20px;'>
							<div>
								<%=rb.getString("LiXianSheBei")%>
								<input id="offlineRetryEnable" name="offlineRetryEnable" class="easyui-checkbox border-box border" type="checkbox" style="width:14px;vertical-align:middle;"/>
								<%=rb.getString("DengDaiSheBeiShangXianChongShi")%>
								<input id="offlineRetryWaitTime" name="offlineRetryWaitTime" class="easyui-numberbox border-box border" style="height:26px;width:60px;" min="20" max="10080" value="60"/>
								<%=rb.getString("FenZhongS")%>
							</div>
							<div style="margin:20px 0;">
								<%=rb.getString("ZaiXianSheBei")%>
								<input id="failedRetryEnable" name="failedRetryEnable" class="border-box border" type="checkbox" style="width:14px;vertical-align:middle;"/>
								<%=rb.getString("PeiZhiShiBaiChongShi")%>
								<input id="failedRetryCount" name="failedRetryCount" class="easyui-numberbox border-box border" value="3" style="height:26px;width:50px;" />
								<%=rb.getString("JianGeCiShuChongShi")%>
								<input id="failedRetryWaitTime" name="failedRetryWaitTime" class="easyui-numberbox border-box border" value="5" style="height:26px;width:50px;" />
								<%=rb.getString("FenZhongS")%>
							</div>
						</div>
					</div>
					<div region="south" data-options="border:false,height:81" style="padding: 20px;">
						<div class="" >
							<a href="#" class="linkbutton linkbutton_nowanna" onclick="downloadMMLTemplate()"><span style="padding:0 20px"><%=rb.getString("DaoChuMuBan")%></span></a>
							<a class="linkbutton linkbutton_trend" href="javascript:void(0)" onclick="saveMMLScriptTask(this)"><span><%=rb.getString("QueDing")%></span></a>
							<a class="linkbutton linkbutton_nowanna" href="javascript:void(0)" onclick="cancelCreateMMLScriptTask()"><span><%=rb.getString("QuXiao")%></span></a>
						</div>
					</div>
				</div>
			</div>
		</div>
		
		<!-- 保存任务时，用此表单提交请求，此任务涉及到上传文件，所以用form表单提交 -->
		<form enctype="multipart/form-data" method="post" id="formSaveMMLScriptTask">
			<input name="uploadFile" type="file" style="display: none;">
			<input name="taskName" type="hidden">
			<input name="status" type="hidden">
			<input name="time" type="hidden">
			<input name="periodStartTime" type="hidden">
			<input name="periodEndTime" type="hidden">
			<input name="periodTime" type="hidden">
			<input name="timeZone" type="hidden">
			<input name="offlineRetryEnable" type="hidden">
			<input name="offlineRetryWaitTime" type="hidden">
			<input name="failedRetryEnable" type="hidden">
			<input name="failedRetryCount" type="hidden">
			<input name="failedRetryWaitTime" type="hidden">
		</form>
		<form id="mmlTemplateExport" style="display:none" method="post"
			action="${ctx}/task/MMLScript/exportMMLTemplete.action">
		</form>
	</div>

	<div id="mml_script_task_result">
		<el-dialog  top="30vh" width="800"
			:visible.sync="dlShow" 
			:modal="false"
		>
			<el-ctable ref="list"
				height="600"
				:data="list"
				:rownumber="true"
				:front-pagination="true"
				:pagination="true"
			>
				<el-table-column label="Line" prop="line" width="80"></el-table-column>
				<el-table-column v-if="false" label="MML" prop="mml" show-overflow-tooltip></el-table-column>
				<el-table-column label="<%=rb.getString("JieGuo")%>" prop="msg" show-overflow-tooltip></el-table-column>
			</el-ctable>
		</el-dialog>
	</div>
</div>
<script type="text/javascript">
var mmlSptvm = new Vue({
	el: '#mml_script_task_result',
	data() {
		
		return {
			dlShow: false,
			list: []
		}
	},
	methods: {
		init() {
			var vm = this;
		}
	}
})

function downloadMMLTemplate(){
	$("#mmlTemplateExport").form("submit",{
		onSubmit: function(param){
			var bool = checkParams(param)
			if(!bool) return false;
		}
	});
}
$(function() {
	var ele = $("#exeTime_addMMLScriptTask");
	disableSelectEarlyTime(ele)
	closeLoading();
            var addOrInfoFlag = sessionStorage.getItem('addOrInfoFlag').split(',');
            if(addOrInfoFlag[0] == 'add'){
                //新建任务
                $('#mmlScriptTaskListInfo').text('<%=rb.getString("XinJianRenWu")%>');
                $('.add_mmlScripts').show();
                $('.info_mmlScripts').hide();
            }else if(addOrInfoFlag[0] == 'info'){
                //查看详情
                $('#mmlScriptTaskListInfo').text('<%=rb.getString("XinXi")%>');
                $('.add_mmlScripts').hide();
                $('.info_mmlScripts').show();

                var params = {};
                params.timeZone = timeZone;
                params.taskId = addOrInfoFlag[1];
                axios.post('${ctx}/task/MMLScript/getMMLScriptDetail.action', stringify(params)).then(function(response){
                    var data = response.data;
                    
                    if(data){
                        $("#taskName_info").val(data.task_name);

                        // 离线设备
                        if( data.offlineRetryEnable == 'on'){
                            $('#offlineRetryEnable_info').prop('checked',true);
                        }else {
                            $('#offlineRetryEnable_info').prop('checked',false);
                        }
                        $("#offlineRetryWaitTime_info").val(data.offlineWaitTime);
                        //在线设备
                        if(data.failedRetryEnable == 'on'){
                            $("#failedRetryEnable_info").prop('checked',true);
                        }else {
                            $("#failedRetryEnable_info").prop('checked',false);
                        }
                        $("#failedRetryCount_info").val(data.failedRetryCount);
                        $("#failedRetryWaitTime_info").val(data.failedRetryWaitTime);

                        // 取得执行方式：激活、挂起、定时执行
                        $("#executionMethods_info").find('input[value='+data.create_status+']').prop('checked',true);
                        $("#executionMethods_info :radio").attr('disabled',true);
                        // 如果是定时执行的，取得时间
                        if(data.create_status == "timing") {
                            $("#time_info").val(data.start_time);
                        }
						// 如果是周期执行的，取得时间
                        if(data.create_status == "period") {
							var timeArr = (data.period_end_time || '').split(' ');

                            $("#period_start_info").val(data.period_start_time);
                            $("#period_end_info").val(timeArr[0]);
                            $("#period_time_info").val(timeArr[1]);
                        }
                    }
                });
            }
	$("#taskName_addMMLScriptTask").val("${addTaskName}");
	// 选择MML脚本文件-文件组件-改变事件
	$("#formSaveMMLScriptTask input[name='uploadFile']").bind("change", function(e) {
		var path = e.target.value;
		if(path == ""){//当选择了取消，则去除文本框中文件,此时不给错误提示
			$("#MMLScriptFileInput").val("");
			return;
		}
		$("#MMLScriptFileInput").val(path);
		<%-- 
		var suffix = path.substring(path.lastIndexOf("."));
		if(suffix != ".txt"){//限制只可以上传txt文件
			e.target.value = "";//清空上传文件
			$("#MMLScriptFileInput").val("");
			$.messager.alert(TiShi, "<%=rb.getString("ZhiZhiChiTxtWenJian")%>");
			return;
		}else{
			$("#MMLScriptFileInput").val(path);
		} 
		--%>
		if(fileFormatMatch(path,'txt')){//限制只可以上传txt文件
			$("#MMLScriptFileInput_err").html('<%=rb.getString("QingXianXuanZeWenJian")%>').hide();
		}else{
			//e.target.value = "";//清空上传文件
			//$("#MMLScriptFileInput").val("");
			$("#MMLScriptFileInput_err").html('<%=rb.getString("ZhiZhiChiTxtWenJian")%>').show();
			return;
		}
	});
	
	
	$("#offlineRetryEnable").bind("change", function(e) {
		if($("#offlineRetryEnable").is(":checked") == true ){
			$("#offlineRetryWaitTime").textbox({required:true})
		}else {
			$("#offlineRetryWaitTime").textbox({required:false})
		}
	});
	
	$("#failedRetryEnable").bind("change", function(e) {
		if($("#failedRetryEnable").is(":checked") == true ){
			$("#failedRetryCount").textbox({required:true});
			$("#failedRetryWaitTime").textbox({required:true});
		}else {
			$("#failedRetryCount").textbox({required:false});
			$("#failedRetryWaitTime").textbox({required:false})
		}
	});
	
	
});

<%-- 验证第一步输入 --%>
function validate_addMMLScriptTask() {
	var errFlag = false;
	
	// 1. 验证任务名称是否填写
	if ($("#taskName_addMMLScriptTask").val().trim().length == 0) {
		showMsg('prompt_msg','<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>');
		errFlag = true;
	} else {
		var exist = false;
		$.ajax({
			type: "post",
			url: "${ctx}/task/MMLScript/taskNameExist.action", 
			data: {"taskName": $("#taskName_addMMLScriptTask").val().trim()},
			async: false,
			dataType: 'json',
			success: function(data) {
				if (data["success"]) {
					if (data["message"] == "true") {// 任务名称已存在
						exist = true;
					}
				}
			}
		});
		if (exist) {
			showMsg('prompt_msg','<%=rb.getString("RenWuMingChengYiCunZai")%>');
			errFlag = true;
		}
	}
	
	// 2. 是否选择了文件
	if($("#formSaveMMLScriptTask input[name='uploadFile']")[0].files.length == 0) {
		//$.messager.alert(TiShi, "<%=rb.getString("QingXianXuanZeWenJian")%>");
		$("#MMLScriptFileInput_err").html('<%=rb.getString("QingXianXuanZeWenJian")%>').show();
		errFlag = true;
	}
	
	// 3. 如果选择的任务状态是”定时执行”，则判断是否填写了时间
	if (document.getElementById("timing_addMMLScriptTask").checked
			&& $("#exeTime_addMMLScriptTask").datetimebox("getValue").length == 0) {
		$(".timeError").show();
		errFlag = true;
	}
	// 4、如果选择周期执行，则判断日期是否填写合法
	if(!validPeriodDateTime()) {
		errFlag = true;
	}

	if($("#MMLScriptFileInput_err").is(':visible')){
		errFlag = true;
	}
	
	if($("#offlineRetryEnable").is(":checked") == true && $("#offlineRetryWaitTime").val() == "" ){
		errFlag = true;
	}
	
	if($("#failedRetryEnable").is(":checked") == true){
		
		if($("#failedRetryCount").val() == ""){
			errFlag = true;
		}
		if($("#failedRetryWaitTime").val() == ""){
			errFlag = true;
		}
	}
	
	if(errFlag){
		return false
	}else{
		return true;
	}
	
}

<%-- 控制定时执行的时间输入框启用/禁用 --%>
function setExeTimerEnable_addMMLScriptTask() {
	if (document.getElementById("timing_addMMLScriptTask").checked) {
		$("#exeTime_addMMLScriptTask").datetimebox("enable");
	} else {
		$("#exeTime_addMMLScriptTask").datetimebox("disable");
		$('.timeError').hide();
	}

	if(document.getElementById("period_addMMLScriptTask").checked) {
		$('#periodStartTime_addMMLScriptTask').datebox("enable");
		$('#periodEndTime_addMMLScriptTask').datebox("enable");
		$('#periodTime_addMMLScriptTask').spinner("enable");
	}else {
		$('#periodStartTime_addMMLScriptTask').datebox("disable");
		$('#periodEndTime_addMMLScriptTask').datebox("disable");
		$('#periodTime_addMMLScriptTask').spinner("disable");
		$('.periodTimeError').hide();
	}
}

<%-- 打开选择MML脚本文件的窗口 --%>
function scanClick_selectMMLScript() {
	$("#formSaveMMLScriptTask input[name='uploadFile']").click();
}

<%-- 保存新建的任务 --%>
function saveMMLScriptTask(ele) {
	$('#addMMLScriptTaskSteps').addClass('loading');
	
	if (!validate_addMMLScriptTask()) {
		$('#addMMLScriptTaskSteps').removeClass('loading');

		return;
	}
	$("#formSaveMMLScriptTask input[name='taskName']").val($("#taskName_addMMLScriptTask").val().trim());
	
	// 取得执行方式：激活、挂起、定时执行
	var radios = document.getElementsByName("taskStatus");
	for (var i = 0; i < radios.length; i++) {
		if (radios[i].checked == true) {
			$("#formSaveMMLScriptTask input[name='status']").val($(radios[i]).attr("status"));
			break;
		}
	}
	
	// 如果是定时执行的，取得时间
	if ($("#formSaveMMLScriptTask input[name='status']").val() == "timing") {
		$("#formSaveMMLScriptTask input[name='time']").val($("#exeTime_addMMLScriptTask").datetimebox("getValue"));
	}

	// 如果是周期执行的，取得时间
	if ($("#formSaveMMLScriptTask input[name='status']").val() == "period") {
		$("#formSaveMMLScriptTask input[name='periodStartTime']").val($("#periodStartTime_addMMLScriptTask").datebox("getValue"));
		$("#formSaveMMLScriptTask input[name='periodEndTime']").val($("#periodEndTime_addMMLScriptTask").datebox("getValue"));
		$("#formSaveMMLScriptTask input[name='periodTime']").val($("#periodTime_addMMLScriptTask").spinner("getValue"));
	}
	
	if( $("#offlineRetryEnable").is(":checked") == true){
		$("#formSaveMMLScriptTask input[name='offlineRetryEnable']").val("on");
	}else {
		$("#formSaveMMLScriptTask input[name='offlineRetryEnable']").val("off");
	}
	if( $("#failedRetryEnable").is(":checked") == true){
		$("#formSaveMMLScriptTask input[name='failedRetryEnable']").val("on");
	}else {
		$("#formSaveMMLScriptTask input[name='failedRetryEnable']").val("off");
	}
	
	$("#formSaveMMLScriptTask input[name='offlineRetryWaitTime']").val($("#offlineRetryWaitTime").val());
	$("#formSaveMMLScriptTask input[name='failedRetryCount']").val($("#failedRetryCount").val());
	$("#formSaveMMLScriptTask input[name='failedRetryWaitTime']").val($("#failedRetryWaitTime").val());
	
	//已经点击了确认按钮，禁用“确认按钮”，以防止服务器响应慢而导致重复点击按钮
	/* $(ele).linkbutton('disable'); */
	
	// 提交请求
    $("#formSaveMMLScriptTask [name='timeZone']").val(timeZone);
	
	submitMMLTask();
	/*
	// 校验文件格式
    uploadWithProgress({
    	url:"${ctx}/task/MMLScript/validFile.action",
    	form: document.querySelector("#formSaveMMLScriptTask"),
    	success: function(data){
    		if(data['success']==true) {// 文件格式正常
    			// 提交添加任务请求
    			submitMMLTask();
    		}else if(data['message']){// 文件格式存在异常
				var message = '<h5><%=rb.getString("WenJianGeShiHuoNeiRongBuZhengQue")%>: </h5>';
				
				message += data['message'];

    			$.messager.confirm('<%=rb.getString("QueRen")%>', message, function (r) {
    				if (r) {
    					// 提交添加任务请求
    	    			// submitMMLTask();
    				}
    			}).addClass('normalConfirm');
				$('#addMMLScriptTaskSteps').removeClass('loading');
    		}else {// 文件为空
    			showMsg('error_msg','<%=rb.getString("WenJianNeiRongWeiKong")%>');
				$('#addMMLScriptTaskSteps').removeClass('loading');
    		}
    	}
    });
	*/
}

function submitMMLTask() {
	uploadWithProgress({
		url:"${ctx}/task/MMLScript/addTask.action",
		form: document.querySelector("#formSaveMMLScriptTask"),
		success: function(data){
			if (data["success"]) {
				showMsg('success_msg',data["msg"]);
	        	cancelCreateMMLScriptTask();
	        	$("#MMLScriptTaskList").datagrid("reload");
	        } else {
				if(data['type'] == 'list') {
					// 队列数据展示
					mmlSptvm.list = data['msg'] || [];
					mmlSptvm.dlShow = true;
				} else {
	        		showMsg('error_msg',data["msg"]);
				}
	        }
			$('#addMMLScriptTaskSteps').removeClass('loading');
		}
	});
}

// 取消新建任务
function cancelCreateMMLScriptTask() {
	$("#winAddMMLScriptTask").slideUp();
            if(sessionStorage.getItem('addOrInfoFlag')){
                sessionStorage.removeItem('addOrInfoFlag');
            }
	/* closeDefaultWindow(); */
}
function selectTimeMML(){
	$('.timeError').hide();
}

function selectStartTimeMML(date) {
	var endDate = $('#periodEndTime_addMMLScriptTask').datebox('getValue');

	validPeriodDateTime();
}
function selectEndTimeMML(date) {
	var startDate = $('#periodStartTime_addMMLScriptTask').datebox('getValue');

	validPeriodDateTime();
}
function selectPeriodTimeMML() {
	validPeriodDateTime();
}

function validPeriodDateTime() {
	var time = $('#periodTime_addMMLScriptTask').spinner('getValue'),
		startDate = $('#periodStartTime_addMMLScriptTask').datebox('getValue'),
		endDate = $('#periodEndTime_addMMLScriptTask').datebox('getValue'),
		startSeconds = new Date(startDate).getTime(),
		endSeconds = new Date(endDate).getTime(),
		bool = true;

	var checked = document.getElementById("period_addMMLScriptTask").checked;

	if(checked) {
		if(time && startSeconds && endSeconds) {
			if(startSeconds - endSeconds >= 0) {
				bool = false;
			}
		}else {
			bool = false;
		}

		if(bool) {
			$('.periodTimeError').hide();
		}else {
			$('.periodTimeError').show();
		}
	}else {
		$('.periodTimeError').hide();
	}

	return bool;
}
</script>
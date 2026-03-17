<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#addMrTask .el-icon-time{
	line-height:1;
}
#addMrTask .timeItem .el-input__inner{
	width:220px;
}
#addMrTask .ml45{
	margin-left: 45px
}
#addMrTask .mt20{
	margin-top: 20px
}
#addMrTask .taskInput{
	width:680px;
	height:28px;
	line-height:28px;	
}
#addMrTask .el-form-item__label{
    height: 32px;
	line-height:32px;
	width:140px;
	text-align:left;
}
#addMrTask .modeItem .el-radio{
	display:inline-block;
	margin-left:0px;
} 
#addMrTask .modeItem{
	margin-top:30px;
}
#addMrTask .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#addMrTask .titleStyML{
	margin-left: 20px;
}
#addMrTask .editButton{
	padding:0 10px;
	height:24px;
	background:#F2F9FF;
	border-radius:2px;
	line-height:24px;
	cursor:pointer;
	margin-left:10px;
	border:1px solid #1DA3FC;
	display:inline-block;
	position:absolute;
	right:130px;
	top:-6px;
}
#addMrTask .editButton i{
	font-size:14px !important;
}
#addMrTask .editButton span{
	font-size:12px;
}
.enbDialogStyle .el-icon-circle-info:before{
	color:#CFCFCF;
}
#addMrTask .el-pairgrid-title{
	top:-6px;
}
#addMrTask .selectItemFormCls{
    width: 40%;
    display:inline-block;
}
#addMrTask .selectItemFormCls .el-input.el-input--small{
    width: 200px;
}
.selectItemFormCls .el-input--suffix .el-input__inner{
    padding-right: 12px;
}
.timeItemCls .el-picker-panel__footer .el-button--text {
    display: none;
}
</style>

<!-- 新建eNB重启任务 -->
<div id="addMrTask" style="margin:20px;">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" label-position="left">
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' prop='taskName' class="ml45 mt20" label-width="120px">
			<el-input maxlength=50 v-model="ruleForm.taskName" size="small" class="taskInput" :disabled="opType == 'view'"></el-input>
		</el-form-item>
        <div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("CeLiangSheZhi")%></span>
		</div>
		<el-form-item prop="measureType" label='<%=rb.getString("CeLiangLeiXing")%>' class="ml45 mt20" label-width="120px">
			<el-checkbox-group v-model="ruleForm.measureType" :disabled="opType == 'view'">
                <el-checkbox label="MRS" disabled>MRS</el-checkbox>
                <el-checkbox label="MRE" disabled>MRE</el-checkbox>
                <el-checkbox label="MRO" disabled>MRO</el-checkbox>
                <el-checkbox label="MDT">MDT</el-checkbox>
            </el-checkbox-group>
		</el-form-item>
        <div class="ml45 mt20" style="display: flex;">
            <el-form-item prop="measurePeriod" label='<%=rb.getString("CeLiangZhouQi")%>'  label-width="120px" class="selectItemFormCls">
                <el-select v-model="ruleForm.measurePeriod" size="small" style="width: 200px;" :disabled="opType == 'view'">
                    <el-option v-for="item in measurePeriodOptions" :key="item.id" :label="item.text" :value="item.id"></el-option>
                </el-select>
            </el-form-item>
            <el-form-item prop="reportPeriod" label='<%=rb.getString("ShangBaoZhouQi")%>' label-width="120px" class="selectItemFormCls">
                <el-select v-model="ruleForm.reportPeriod" @change="reportPeriodChange" size="small" style="width: 200px;" :disabled="opType == 'view'">
                    <el-option label="15min" value="15"></el-option>
                    <el-option label="30min" value="30"></el-option>
                    <el-option label="60min" value="60"></el-option>
                </el-select>
            </el-form-item>
        </div>
        <div class="ml45 mt20" style="display: flex;">
            <el-form-item prop="startDataTime" label='<%=rb.getString("KaiShiShiJian")%>'  label-width="120px" class="selectItemFormCls">
                <el-date-picker 
                    :disabled="opType == 'view'"
                    popper-class="timeItemCls"
                    style="width: 120px;"
                    size="small"
                    v-model="ruleForm.startData"
                    type="date"
                    value-format="yyyy-MM-dd"
                    format="yyyy-MM-dd"
                    :clearable="false"
                    :picker-options="dataPickerOptions"
                    >
                </el-date-picker>
                <span>-</span>
                <el-time-select
                    :disabled="opType == 'view'"
                    style="width: 80px;"
                    size="small"
                    v-model="ruleForm.startTime"
                    :picker-options="timePickerOptions"
                    :clearable="false"
                    >
                </el-time-select>
            </el-form-item>
            <el-form-item prop="endDataTime" label='<%=rb.getString("JieShuShiJian")%>'  label-width="120px" class="selectItemFormCls">
                <el-date-picker 
                    :disabled="opType == 'view' || ruleForm.unlimitedTime"
                    popper-class="timeItemCls"
                    style="width: 120px;"
                    size="small"
                    v-model="ruleForm.endData"
                    type="date"
                    value-format="yyyy-MM-dd"
                    format="yyyy-MM-dd"
                    :clearable="false"
                    :picker-options="dataPickerOptions"
                    >
                </el-date-picker>
                <span>-</span>
                <el-time-select
                    :disabled="opType == 'view' || ruleForm.unlimitedTime"
                    style="width: 80px;"
                    size="small"
                    v-model="ruleForm.endTime"
                    :picker-options="timePickerOptions"
                    :clearable="false"
                    >
                </el-time-select>
                <el-checkbox v-model="ruleForm.unlimitedTime" @change="unlimitedTimeChange" size="small" style="margin-left: 10px;" :disabled="opType == 'view'"></el-checkbox>
                <span style="margin-left: 5px;color: #aaa;"><%=rb.getString("BuXianZhiShiJian")%></span>
            </el-form-item>
        </div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<el-pairgrid v-if="showPairGrid" :id="'enb_select_list'" ref="cpairgrid" @selection-change='selectChange'  :right-url="rightUrl" :left-url="leftUrl" :height="height" row-key="smallCellCode" :query-params="queryParams" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" style='margin:10px 45px 0px 45px;'>
			<template slot="left">
				<el-table-column type="selection" width="45"></el-table-column>
				<el-table-column prop="connection_status" width="50">
					<template slot-scope="scope">
						<div :class="{
							'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
							'':scope.row.have_connected==2,
							'conn_exc':scope.row.connection_status=='Exception',
							'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
					</template>
				</el-table-column>
				<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
				<el-table-column prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
			</template>
			<template slot='toolbar'>
			<!-- 新建任务 基站列表的高级选择 -->
				<el-form :model='queryForm' ref="queryForm" label-position="top">
					<div style="display: flex;align-items: center;padding: 10px 0px;">
						<el-query type="normal" @query="query"  :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"></el-query>
						<el-popfilter style="margin: 0 5px;"
							type="single"
							label='<%=rb.getString("SheBeiZu")%>'
							v-model="queryForm.groupId"
							:list="groupOptions.map(item=>{return {label:item.group_name,value:item.id}})"
							@check-change="advanceQuery">
						</el-popfilter>
						<div class="pop-filter-clear" style="margin: 0 5px;" 
							@click="resetQuery">
							<%=rb.getString("QingKongShaiXuan")%>
						</div>
					</div>
				</el-form>
				<div class="editButton" size="mini" @click="addBatchSn">
					<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
					<span><%=rb.getString("PiLiangShuRu")%></span>
				</div>
			</template>
			<template slot='right'>
				<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
				<el-table-column prop='hostName' label='<%=rb.getString("HostName")%>'></el-table-column>
			</template>
		</el-pairgrid>
		<div v-else style='margin:0px 45px;'>
			<div style='font-size:14px;padding-top:10px;'><%=rb.getString("YiXuanZeJiZhan")%></div>
			<el-ctable :id="'selected_device_list'" ref="stable"  :url="rightUrl" :height="height" pagination="true" style="border: 1px solid #F3F3F3;">
				<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serialNumber"></el-table-column>
				<el-table-column label='<%=rb.getString("HostName")%>' prop="hostName"></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='smallCellStr' style="margin-left:45px;margin-bottom:30px;">
			<el-input v-model='ruleForm.smallCellStr' v-show="false"></el-input>
		</el-form-item>
	</el-form>
	<el-dialog class='enbDialogStyle' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="top">
			<div>
				<label><%=rb.getString("XiaoZhanBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='addListForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">
var addMrTaskVue = new Vue({
	el:'#addMrTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/cell/perfmgmt/mrcustomize/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskName.trim()
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						if(data["message"] == "true"){
							callback(new Error('<%=rb.getString("RenWuMingChengYiCunZai")%>'))
						}else{
							callback();
						}
					}
				}).catch(function(error){
					callback()
				})
			}
		};
		var validateStartDateTime = (rule,value,callback) => {
			if(value == '' || value==null){
                callback()
            }else{
                if(this.ruleForm.unlimitedTime){
                    callback()
                }else{
                    const startTime = this.ruleForm.startDataTime;
                    const endTime = this.ruleForm.endDataTime;
                    const startDate = new Date(startTime.replace(' ', 'T'));
                    const endDate = new Date(endTime.replace(' ', 'T'));
                    if (startDate >= endDate) {
                    callback(new Error('<%=rb.getString("JieShuShiJianBuNengXiaoYuKaiShiShiJian")%>'));
                    } else {
                    callback();
                    }
                }
            }
		}
		var validatorNum = (rule,value,callback) => {
			var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
				list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
					return item.length > 0;
				});
			
			if (serialNumber == null || serialNumber.length == 0) {
				callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
			} else {
				var nameFlag = list.every(function(item,index){
					return temp.test(item)
				})
				if(nameFlag){
					callback()
				}else{
					callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
				}
			}
		};
		return {
            opType: '',
			leftUrl:'',
			rightUrl:'',
			height:'370px',
			groupOptions:[],
            measurePeriodOptions:[
                {id:'2048',text:'ms2048'},
                {id:'5120',text:'ms5120'},
                {id:'10240',text:'ms10240'},
                {id:'1',text:'min1'},
                {id:'6',text:'min6'},
                {id:'12',text:'min12'},
                {id:'30',text:'min30'},
                {id:'60',text:'min60'},
            ],
			deviceTitle:['<%=rb.getString("JiZhanLieBiao")%>','<%=rb.getString("YiXuan")%>'],
			rowData : [],
			selection:'',
			ruleForm:{
				taskName:'',
                measureType:['MRS','MRE','MRO'],
                measurePeriod:'5120',
                reportPeriod:'15',
                startDataTime: '',
                startData: '',
                startTime: '',
                endDataTime: '',
                endData: '',
                endTime: '',
                unlimitedTime:false,
				smallCellStr:'',
                hostNameStr:'',
                serialNumberStr:'',
			},
			rules:{
				taskName:[
					{validator:validateName,trigger:'blur'}
				],
                startDataTime:[
                    {validator:validateStartDateTime,trigger:'change'}
                ],
				smallCellStr:[
					{required:true,message:'<%=rb.getString("QingXuanZeSheBei")%>',trigger:'change'}
				],
			},
			queryParams:{
				timeZone:timeZone,
                like_field:'serial_number,host_name',
                isUseTemplate: 'true',
				searchText:'',
				groupId:'',
			},
			queryForm:{
				groupId:'',
			},
			taskId:'',
			defaultTaskName:'',
			addListForm:{
				serialNumber:''
			},
			addListRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'change'}
				]
			},
			listVisible:false,
			dialogTitle:'<%=rb.getString("TianJia")%>',
            dataPickerOptions: {
                disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
            },
            showPairGrid: true
		}
	},
    computed: {
        timePickerOptions() {
            let params ={
                start: '00:00',
                step: '00:15',
                end: '23:45'
            }
            if(this.ruleForm.reportPeriod == '30'){
                params.step = '00:30'
                params.end = '23:30'
            }
            if(this.ruleForm.reportPeriod == '60'){
                params.step = '00:60'
                params.end = '23:00'
            }
            return params
        },
        startDataTime(){
            var dataTimeStr = this.ruleForm.startData + ' ' + this.ruleForm.startTime;
            return dataTimeStr
        },
        endDataTime(){
            var dataTimeStr = this.ruleForm.endData + ' ' + this.ruleForm.endTime;
            return dataTimeStr
        }
    },
    watch:{
		selection(){
			var data = this.$refs.cpairgrid.getData();
			var smallCellCodeList = [],
                hostNameList = [],
                serialNumberList = [];
			if(data.length != 0){
				data.map(function(item){
                    smallCellCodeList.push(item.smallCellCode);
                    hostNameList.push(item.hostName);
                    serialNumberList.push(item.serialNumber);
				})
			}

			this.ruleForm.smallCellStr = smallCellCodeList.join(',');
            this.ruleForm.hostNameStr = hostNameList.join(',');
            this.ruleForm.serialNumberStr = serialNumberList.join(',');
		},
        startDataTime(){
            this.ruleForm.startDataTime = this.startDataTime + ':00'
        },
        endDataTime(){
            this.ruleForm.endDataTime = this.endDataTime + ':00'
            this.$refs.ruleForm.validateField('startDataTime')
        }
	},
	methods:{ 
        // 初始化
		init(row,type){
			var vm = this;
            vm.opType = type;
			axios.post('${ctx }/cell/cpeinfos/getDeviceGroupListByCell.action').then(function(response){
				let data = response.data;
                let arrList = [{group_name:'<%=rb.getString("QuanBu")%>',id:''}];
                vm.groupOptions = arrList.concat(data ? data : []);
			}).catch(function(error){})
            vm.leftUrl = '${ctx}/pm/template/getEnbListPageData.action';
            if(vm.opType == 'add'){
                vm.ruleForm.taskName = 'MR'+ '<%=rb.getString("RenWu")%>' + '_' + user_code + '_' + formatDate(new Date(gloableTime));
                vm.ruleForm.startData = vm.getCurentDateStr();
                vm.ruleForm.endData = vm.getCurentDateStr();
                vm.ruleForm.startTime = vm.getCurentTimeStr('start',vm.ruleForm.reportPeriod);
                vm.ruleForm.endTime = vm.getCurentTimeStr('end',vm.ruleForm.reportPeriod);
            }else{
                vm.showPairGrid = false;
                vm.rightUrl = '${ctx}/cell/perfmgmt/mrcustomize/getMRTaskproperty.action?taskId=' + row.task_id;
                vm.ruleForm.taskName = row.task_name;
                vm.ruleForm.measureType = row.mr_type.split(',');
                vm.ruleForm.measurePeriod = row.statis_period + '';
                vm.ruleForm.reportPeriod = row.report_period + '';
                vm.ruleForm.startData = row.start_time.split(' ')[0];
                vm.ruleForm.startTime = row.start_time.split(' ')[1].slice(0,5);
                if(row.end_time){
                    vm.ruleForm.endData = row.end_time.split(' ')[0];
                    vm.ruleForm.endTime = row.end_time.split(' ')[1].slice(0,5);
                }else{
                    vm.ruleForm.unlimitedTime = true;
                    vm.ruleForm.endData = '';
                    vm.ruleForm.endTime = '';
                }
            }
            vm.$nextTick(function(){
                initForm(vm.$refs.ruleForm);
            })
		},
        // 步长改变
        reportPeriodChange(){
            this.ruleForm.startTime = this.getCurentTimeStr('start',this.ruleForm.reportPeriod);
            this.ruleForm.endTime = this.getCurentTimeStr('end',this.ruleForm.reportPeriod);
        },
		/**
		 * 模糊查询
		 * @param val:查询参数
		*/
		query(val){
			this.queryParams.searchText = val;
			this.$refs.cpairgrid.reload();
		},
		advanceQuery(){ // 高级搜索查询
			var vm = this;
			Object.assign(vm.queryParams,vm.queryForm);
		},
		resetQuery(){ // 普通搜索查询按钮
			var vm = this,
				params = {
					groupId:'',
				};
			Object.assign(vm.queryForm,params);
			Object.assign(vm.queryParams,params);
		},
		/**
		 * 全选操作
		 * @param selection:选择的数据
		*/
		selectChange(selection){
			this.selection = selection
		},
        // 不限制时间按钮改变事件
        unlimitedTimeChange(){
            var vm = this;
            vm.$refs.ruleForm.validateField('startDataTime')
        },
		submit(){ // 确定按钮 
	    	var vm = this,
                urls = '${ctx}/cell/perfmgmt/mrcustomize/addMRTask.action',
                params = {
                    timeZone:timeZone,
                    taskName:vm.ruleForm.taskName,
                    type:vm.ruleForm.measureType.join(','),
                    measurePeriod:vm.ruleForm.measurePeriod,
                    reportPeriod:vm.ruleForm.reportPeriod,
                    startDate:vm.ruleForm.startDataTime,
                    endDate:vm.ruleForm.endDataTime,
                    task_status: 'on',
                    smallCellStr: vm.ruleForm.smallCellStr,
                    hostNameStr: vm.ruleForm.hostNameStr,
                    serialNumberStr: vm.ruleForm.serialNumberStr,
                };
            if(vm.ruleForm.unlimitedTime){
                params.unlimitedTime = 'on'
                delete params.endDate
            }
            // 防止多次提交
            if(mrPageVue.slideSubmitLoading)return

	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    	    	mrPageVue.slideSubmitLoading = true;
	    			axios.post(urls,stringify(params)).then(function(response){
	    				var data = response.data;
	    				if(data["success"]){
	    					vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success',
	    					})
                            eventBus.$emit('hide-addSlide');
	    				}else{
	    					vm.$message.error(data["message"]);
                            mrPageVue.slideSubmitLoading = false;
	    				}
	    			})
	    		}else{
	    			return false;
	    		}
	    	})
		},
		cancel(){ // 关闭弹窗
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
			if(isFormChanged(this.$refs.ruleForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					eventBus.$emit('hide-addSlide')
				}).catch(() => {
					
				})
			}else{
				eventBus.$emit('hide-addSlide')
			}
		},
		addBatchSn(){
			this.listVisible = true;
		},
		closeBatchSn(){
			this.listVisible = false;
			this.$refs.addListForm.resetFields();
		},
		saveBatchSn(){
			var vm = this, snStr = vm.addListForm.serialNumber || '',
			list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			
			var params = {
				serialNumbers : list.join(","),
			}
			vm.$refs.addListForm.validate((valid) => {
				if(valid){
					axios.post("${ctx}/cell/perfmgmt/mrcustomize/getMRSelectedDevices.action",stringify(params)).then((res)=>{
						var data = res.data;
						if(data && data.length > 0){
							vm.$refs.cpairgrid.appendCheckedRows(data);
							vm.closeBatchSn();
						}else{
							vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>');
						}
					})
				}
			})
		},
        // 获取当前日期
        getCurentDateStr(){
            var now = new Date(gloableTime);
            var year = now.getFullYear();
            var month = now.getMonth()+1;
            var day = now.getDate();
            var clock = year + "-";
            if(month < 10) clock += "0";
            clock += month + "-";
            if(day <10) clock += "0";
            clock += day;
            
            return clock;
        },
        getCurentTimeStr(type,reportPeriod){
            var timeStr = '';
            var currDate = new Date(gloableTime);
            var periodVal = reportPeriod;
            var h = currDate.getHours();
            var m = currDate.getMinutes();
            var startMins = periodVal * ( parseInt( m / periodVal ) +1 );
    
            if( startMins == 60){
                h++;
                m = '00';
            }else{
                m = startMins
            }
            if(type == 'start'){
                timeStr =  h + ':' + m;
            }else{
                timeStr =  (h + 1)+ ':' + m;
            }
            return timeStr;
        }
	},
	
	mounted(){
		eventBus.$off('init-addMr').$on('init-addMr',this.init);
		eventBus.$off('addMrTask-submit').$on('addMrTask-submit',this.submit);
		eventBus.$off('addMrTask-cancel').$on('addMrTask-cancel',this.cancel);
	}
})
</script>
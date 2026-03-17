<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	.url-list {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
	}
	.url-list > span {
		position: relative;
		display: inline-block;
		height: 12px;
		border: 1px dashed #DCDFE6;
		border-radius: 3px;
		padding: 5px;
		margin: 5px 20px 5px 0px;
		background: #F5F7FA;
		font-weight: bold;
		color: #666;
	}
	.url-bt {
		position: absolute;
		zoom: 0.8;
		top: -7px;
		right: -7px;
	}

	.url-span.active {
		border: 1px dashed #4D84FF;
		color: #4D84FF;
	}
</style>
<div id='diagnosticPage' class='diagnosticPage'>
	<div class='oper-area'>
		<el-select v-model="testType" class='typeCls' :disabled="typeLimit">
			<el-option v-for="item in testList" :key="item.value" :value="item.value" :label="item.label"></el-option>
		</el-select>
		<span v-show="urlHeadFlag" class='urlHead'>http://</span>
		<el-input placeholder="<%=rb.getString("ShuRuFuWuQiDiZhi")%>" v-model="server_url" clearable style='width:500px;'>
			<!-- <template slot="prepend" v-if='urlHeadFlag'>http://</template> -->
		</el-input>
		
		<span class='sizeTitle' v-show="upLimit"><%=rb.getString("CeSuWenJianDaXiao")%></span>
		<el-select v-model="fileSize" class='sizeCls' v-show="upLimit">
			<el-option v-for="item in fileSizeList" :key="item.value" :value="item.value" :label="item.label"></el-option>
		</el-select>

		<span class='sizeTitle' v-show="testType == 'PING'"><%=rb.getString("TongJiCiShu")%></span>
		<el-select v-model="pingNum" class='pingCls' v-show="testType == 'PING'">
			<el-option v-for="item in pingCountList" :key="item.value" :value="item.value" :label="item.label"></el-option>
		</el-select>

		<el-button type="primary" class='startButton' @click="startTest">{{buttonTitle}}</el-button>
		<p class='url-item-cls' v-if="false">
			<span v-for="item in urlList" @click="clickUrl(item)">{{item}}</span>
		</p>
	</div>
	<div style="margin: 10px 50px;padding: 5px;border: 1px solid #DCDFE6;border-radius: 3px;display: flex;align-items: center;">
		<div style="padding: 10px; min-width: 125px;height: 15px;">
			Common URLs  
			<span style="display: inline-block;padding-left: 10px;;color: silver;cursor: pointer;" @click="clearHis">Clear</span>
		</div>
		<div class="url-list">
			<span class="url-span" v-if="testType == 'DOWNLOAD'" v-for="(item, idx) in commonUrl.DOWNLOAD">
				<span @click="reviewUrl(item, event)">{{item}}</span> 
				<i class="el-icon el-icon-close url-bt" @click="removeHis(idx)"></i>
			</span>
			<span class="url-span" v-if="testType == 'UPLOAD'" v-for="(item, idx) in commonUrl.UPLOAD">
				<span @click="reviewUrl(item, event)">{{item}}</span> 
				<i class="el-icon el-icon-close url-bt" @click="removeHis(idx)"></i>
			</span>
			<span class="url-span" v-if="testType == 'PING'" v-for="(item, idx) in commonUrl.PING">
				<span @click="reviewUrl(item, event)">{{item}}</span> 
				<i class="el-icon el-icon-close url-bt" @click="removeHis(idx)"></i>
			</span>
			<span class="url-span" v-if="testType == 'TRACE'" v-for="(item, idx) in commonUrl.TRACE">
				<span @click="reviewUrl(item, event)">{{item}}</span> 
				<i class="el-icon el-icon-close url-bt" @click="removeHis(idx)"></i>
			</span>
		</div>
	</div>
	<div class='result-area'>
		<div class='result-item' :class="testType == 'DOWNLOAD' ? 'testStyle' : ''" v-loading="downLoading">
			<div class='item-header'>
				DownloadTest
			</div>
			<div class='item-body'>
				<div id='downloadPanel' style='width:400px;height:300px;'></div>
				<div style='flex:1;height:280px;border:1px solid #F3F3F3;position:relative'>
					<div label="<%=rb.getString("CeShiShiJian")%>" class='item-title'>{{downList.start_time}}</div>
					<div label="<%=rb.getString("CeSuFuWuQi")%>" class='item-title'>{{downList.download_url}}</div>
					<div label="<%=rb.getString("ZhuangTai")%>" class='item-title'>{{downList.status}}</div>
					<div label="<%=rb.getString("CeSuLianJieShiJian")%>(ms)" class='item-title'>{{downList.connect_time}}</div>
					<div label="<%=rb.getString("CeSuXiaZaiShiJian")%>(ms)" class='item-title'>{{downList.download_time}}</div>
					<div label="<%=rb.getString("CeSuWenJianDaXiao")%>(MB)" class='item-title'>{{downList.file_size}}</div>
					<div label="<%=rb.getString("CeSuSuDu")%>(Mbps)" class='item-title'>{{downList.speed}}</div>
					<div label="" class='item-title'></div>
				</div>
			</div>
		</div>
		<div class='result-item' :class="testType == 'UPLOAD' ? 'testStyle' : ''" v-loading="upLoading">
			<div class='item-header'>
				UploadTest
			</div>
			<div class='item-body'>
				<div id='uploadPanel' style='width:400px;height:300px;'></div>
				<div style='flex:1;height:280px;border:1px solid #F3F3F3;position:relative'>
					<div label="<%=rb.getString("CeShiShiJian")%>" class='item-title'>{{upList.start_time}}</div>
					<div label="<%=rb.getString("CeSuFuWuQi")%>" class='item-title'>{{upList.upload_url}}</div>
					<div label="<%=rb.getString("ZhuangTai")%>" class='item-title'>{{upList.status}}</div>
					<div label="<%=rb.getString("CeSuLianJieShiJian")%>(ms)" class='item-title'>{{upList.connect_time}}</div>
					<div label="<%=rb.getString("CeSuShangChuanShiJian")%>(ms)" class='item-title'>{{upList.upload_time}}</div>
					<div label="<%=rb.getString("CeSuWenJianDaXiao")%>(MB)" class='item-title'>{{upList.file_size}}</div>
					<div label="<%=rb.getString("CeSuSuDu")%>(Mbps)" class='item-title'>{{upList.speed}}</div>
					<div label="" class='item-title'></div>
				</div>
			</div>
		</div>
		<div class='result-item' :class="testType == 'PING' ? 'testStyle' : ''" v-loading="pingLoading">
			<div class='item-header'>
				PingTest
			</div>
			<div class="item-body" style='padding-left:50px;padding-top:20px;'>
				<div label="<%=rb.getString("CeShiShiJian")%>" class='ping-title'>{{pingList.start_time}}</div>
				<div label="<%=rb.getString("CeSuFuWuQi")%>" class='ping-title'>{{pingList.host}}</div>
				<div label="<%=rb.getString("ZhuangTai")%>" class='ping-title' style='border-right:none'>{{pingList.status}}</div>
				<div style='width:90%;background:#E9E9E9;height:1px;'></div>
				<div label="<%=rb.getString("CeSuChengGongShu")%>" class='ping-title'>{{pingList.success_count}}</div>
				<div label="<%=rb.getString("CeSuShiBaiShu")%>" class='ping-title'>{{pingList.failure_count}}</div>
				<div label="<%=rb.getString("PingJunXiangYingShiJian")%>" class='ping-title' style='border-right:none'>{{pingList.average_response_time}}</div>
			</div>
		</div>
		<div class='result-item' :class="testType == 'TRACE' ? 'testStyle' : ''" v-loading="traceLoading">
			<div class='item-header'>
				Traceroute
			</div>
			<div class='item-body'>
				<el-ctable ref='traceTable' :data="traceData" height="100%" style='width:100%'>
					<template slot="toolbar">
						<div label="<%=rb.getString("CeShiShiJian")%>" class='trace-title'>{{traceList.start_time}}</div>
						<div label="<%=rb.getString("CeSuFuWuQi")%>" class='trace-title'>{{traceList.host}}</div>
						<div label="<%=rb.getString("ZhuangTai")%>" class='trace-title' style='border-right:none'>{{traceList.status}}</div>
					</template>
					<el-table-column label='HopHost' prop="hop_host" ></el-table-column>
					<el-table-column label='HopHostAddress' prop="hop_host_address" ></el-table-column>
					<el-table-column label='HopErrorCode' prop="hop_error_code" ></el-table-column>
					<el-table-column label='HopRTTimes' prop="hop_rt_time" show-overflow-tooltip=true></el-table-column>
				</el-ctable>
			</div>
		</div>
	</div>
</div>
<script>
	var timer;
	var testVm = new Vue({
		el:"#diagnosticPage",
		data(){
			return{
				navActive: 'DOWNLOAD',
				commonUrl: {
					DOWNLOAD: [],
					UPLOAD: [],
					PING: [],
					TRACE: []
				},
				
				testType:'DOWNLOAD',
				server_url:'',
				testList:[
					{label:'DownloadTest',value:'DOWNLOAD'},
					{label:'UploadTest',value:'UPLOAD'},
					{label:'PingTest',value:'PING'},
					{label:'Traceroute',value:'TRACE'}
				],
				urlList:[],
				traceData:[],
				upLimit:false,
				cpeCode:'',
				fileSize:'',
				pingNum: 1,
				downList:{
					start_time:'',
					download_url:'',
					connect_time:'',
					download_time:'',
					file_size:'',
					speed:'',
					status:''
				},
				upList:{
					start_time:'',
					upload_url:'',
					connect_time:'',
					upload_time:'',
					file_size:'',
					speed:'',
					status:''
				},
				pingList:{
					start_time:'',
					status:'',
					host:'',
					success_count:'',
					failure_count:'',
					average_response_time:''
				},
				traceList:{
					start_time:'',
					status:'',
					host:''
				},
				downLoading:false,
				upLoading:false,
				pingLoading:false,
				traceLoading:false,
				buttonTitle:'Start',
				typeLimit:false,
				serialNumber:'',
				cellName:'',
				fileSizeList:[
					{value:5,label:'5M'},
					{value:10,label:'10M'},
					{value:100,label:'100M'},
					{value:500,label:'500M'},
					{value:600,label:'600M'},
					{value:700,label:'700M'},
					{value:800,label:'800M'},
					{value:900,label:'900M'},
					{value:1000,label:'10000M'}
				],
				pingCountList:[
					{value:1,label:'1'},
					{value:2,label:'2'},
					{value:3,label:'3'},
					{value:4,label:'4'},
					{value:5,label:'5'},
					{value:6,label:'6'},
					{value:7,label:'7'},
					{value:8,label:'8'},
					{value:9,label:'9'},
					{value:10,label:'10'}
				],
				urlHeadFlag:true
			}
		},
		watch:{
			"testType":function(val){
				this.upLimit = val == 'UPLOAD' ? true : false;
				if(val == 'PING' || val == 'TRACE'){
					this.urlHeadFlag = false;
				}else{
					this.urlHeadFlag = true;
				}

				this.getServer();
			},
			"downList.status":function(val){
				this.clearTimer(val,'downLoading');
			},
			"upList.status":function(val){
				this.clearTimer(val,'upLoading');
			},
			"pingList.status":function(val){
				this.clearTimer(val,'pingLoading');
			},
			"traceList.status":function(val){
				this.clearTimer(val,'traceLoading');
			},
			buttonTitle(val){
				this.typeLimit = val == 'Start' ? false : true;
			},
			commonUrl: {
				handler(map) {
					var vm = this,
						type = vm.testType,
						urllist = map[type],
						params = {
							operatorCode: operatorCodeGloab,
							commnUrls: urllist.join(','),
							type: type
						};

					axios.post("${ctx}/cpe/diagnostics/saveUrls.action",stringify(params)).then(res=>{})
				},
				deep: true
			}
		},
		methods:{
			init(code){
				this.cpeCode = code;
				this.getTestData('DOWNLOAD');
				this.getTestData('UPLOAD');
				this.getTestData('PING');
				this.getTestData('TRACE');
				this.getServer();
			},
			reviewUrl(url, evt) {
				var vm = this;

				vm.server_url = url.replace('http://','');
				$('.url-span').removeClass('active');
				$(evt.target).parent().addClass('active');
			},
			removeHis(idx) {
				var vm = this,
					type = vm.testType;
				
				vm.commonUrl[type].splice(idx,1);
			},
			addHis(url) {
				var vm = this,
					trimedUrl = url.trim(),
					type = vm.testType;
				
				if(!vm.commonUrl[type].includes(trimedUrl) && vm.commonUrl[type].length<5) {
					vm.commonUrl[type].push(trimedUrl);
				}
			},
			clearHis() {
				var vm = this,
					type = vm.testType;
				
				vm.commonUrl[type] = [];
			},
			clearTimer(val,type){
				var vm = this;
				if(!(val == 'None' || val == 'Requested' || val == 'In Progress' || val == 'Stopping')){
					clearInterval(timer);
					vm[type] = false;
					vm.buttonTitle = 'Start';
				}
			},
			initDownload(val){
				var vm = this;
				var chartDom = document.getElementById('downloadPanel');
				var myChart = echarts.init(chartDom);
				var color;
				if(val == 0 || val == null || val == undefined){
					color = '#ECF0FA';
					val = 0;
				}else{
					color = '#19D5F3';
				}
				var option = {
						series:[{
							type:'gauge',
							max:1000,
							itemStyle:{
								color:color
							},
							progress:{
								show:true,
								width:18,
								
							},
							axisLine:{
								lineStyle:{
									width:18,
									color:[[1,'#ECF0FA']]
								}
							},
							axisTick:{
								show:false
							},
							splitLine:{
								length:8,
								distance:5,
								lineStyle:{
									width:1,
									color:'#707070'
								}
							},
							axisLabel:{
								distance:25,
								color:'#333',
								fontSize:12
							}, 
							anchor:{
								show:true,
								showAbove:true,
								size:23,
								itemStyle:{
									color:color
								}
							},
							pointer:{
								length:'50%',
							},
							detail:{
								valueAnimate:true,
								fontSize:36,
								offsetCenter:[0,'70%']
							},
							title:{
								show:true,
								offsetCenter:[0,'110'],
								color:'#333',
								fontSize:18
							},
							data:[{
								value:val,
								name:'Mbps',
							}]
						}]
				}
				option && myChart.setOption(option);
			},
			initUpload(val){
				var vm = this;
				var chartDom = document.getElementById('uploadPanel');
				var myChart = echarts.init(chartDom);
				var color = val == 0 ? '#ECF0FA' : '#19D5F3';
				var option = {
						series:[{
							type:'gauge',
							max:1000,
							itemStyle:{
								color:color
							},
							progress:{
								show:true,
								width:18,
								
							},
							axisLine:{
								lineStyle:{
									width:18,
									color:[[1,'#ECF0FA']]
								}
							},
							axisTick:{
								show:false
							},
							splitLine:{
								length:8,
								distance:5,
								lineStyle:{
									width:1,
									color:'#707070'
								}
							},
							axisLabel:{
								distance:25,
								color:'#333',
								fontSize:12
							}, 
							anchor:{
								show:true,
								showAbove:true,
								size:23,
								itemStyle:{
									color:color
								}
							},
							pointer:{
								length:'50%',
							},
							detail:{
								valueAnimate:true,
								fontSize:36,
								offsetCenter:[0,'70%']
							},
							title:{
								show:true,
								offsetCenter:[0,'110'],
								color:'#333',
								fontSize:18
							},
							data:[{
								value:val,
								name:'Mbps',
							}]
						}]
				}
				option && myChart.setOption(option);
			},
			getTestData(type){
				var vm = this;
				var typeList = {
						DOWNLOAD : 'downList',
						UPLOAD : 'upList',
						PING : 'pingList',
						TRACE : 'traceData'
				}
				var params = {
						cpeCode : vm.cpeCode,
						type : type
				}
				axios.post("${ctx}/cpe/diagnostics/queryDiagnosticsData.action",stringify(params)).then(res=>{
					var data = res.data;
					var speed = 0;
					if(data.rows.length > 0){
						if(type != 'TRACE'){
							Object.assign(vm[typeList[type]],data.rows[0]);
						}else{
							vm[typeList[type]] = data.rows;
							vm.traceList.start_time = data.rows[0].start_time;
							vm.traceList.status = data.rows[0].status;
							vm.traceList.host = data.rows[0].host;
						}
						speed = data.rows[0].speed == null ? 0 : data.rows[0].speed;
						if(data.rows[0].status == 'In Progress'){
							vm.buttonTitle = 'Stop';
							clearInterval(timer);
							timer = setInterval(function(){
								vm.getTestData(vm.testType);
							},6000)
						}
					}
					if(type == 'DOWNLOAD'){
						vm.initDownload(speed);
					}
					if(type == 'UPLOAD'){
						vm.initUpload(speed);
					}
				})
			},
			clickUrl(val){
				this.server_url = val;
			},
			getServer(){
				var vm = this,
					type = vm.testType,
					params = {
						operatorCode: operatorCodeGloab,
						type: type
					};

				axios.post("${ctx}/cpe/diagnostics/queryDiagnosticsServerUrl.action", stringify(params)).then(res=>{
					//vm.urlList = res.data;
					vm.commonUrl[type] = res.data?res.data.split(','):[];
				})
			},
			startTest(){
				var vm = this;
				if(vm.server_url != ''){
					if(vm.testType == 'UPLOAD' || vm.testType == 'DOWNLOAD'){
						var server_url = 'http://' + vm.server_url;
					}else{
						var server_url = vm.server_url;
					}
					var url = vm.buttonTitle == 'Start' ? '${ctx}/cpe/diagnostics/addTask.action' : '${ctx}/cpe/diagnostics/stopTask.action'
					var params = {
						cpeCode : vm.cpeCode,
						url : server_url,
						type : vm.testType,
					}
					if(vm.testType == 'UPLOAD'){
						params.fileLength = vm.fileSize
					}
					
					if(vm.testType == 'PING'){
						params.pingNum = vm.pingNum
					}

					var typeList = {
							'DOWNLOAD' : 'downLoading',
							'UPLOAD' : 'upLoading',
							'PING' : 'pingLoading',
							'TRACE' : 'traceLoading'
					}
					
					vm.addHis(server_url);
					
					axios.post(url,stringify(params)).then(res=>{
						var data = res.data;
						if(data["success"]){
							if(vm.buttonTitle == 'Start'){//开始定时更新
								vm.buttonTitle = 'Stop';
								vm[typeList[vm.testType]] = true;
								timer = setInterval(function(){
									vm.getTestData(vm.testType);
								},6000)
							}else{//结束定时
								vm.buttonTitle = 'Start';
								//clearInterval(timer);
								vm.getTestData(vm.testType);
								vm[typeList[vm.testType]] = false;
							}
						}else{
							vm.$message.error(data.message);
							vm[typeList[vm.testType]] = false;
						}
					})
				}
			}
		},
		mounted(){
			eventBus.$off('cpe-data').$on('cpe-data',this.init)
		}
	})
</script>
<style>
.diagnosticPage{
	display:flex;
	flex-direction:column;
	overflow:hidden;
	height:100%;
	border:1px solid #d5dcec;
	border-radius:10px;
	background:#fff;
	box-sizing:border-box;
}
.diagnosticPage .typeCls .el-input{
	width:145px;
	height:40px;
}
.diagnosticPage .typeCls .el-input .el-input__inner{
	width:145px;
	height:40px;
	font-weight:bold;
	font-size:14px;
	color:#333;
}
.diagnosticPage .el-input-group .el-input__inner{
	height:40px;
	width:500px;
}
.el-input-group{
	margin-left:10px;
}
.oper-area{
	margin:30px auto 0px;
}
.url-item-cls{
	margin-left:160px;
}
.url-item-cls span{
	display:inline-block;
	padding:0px 10px;
	height:20px;
	line-height:20px;
	border:1px dotted #4D84FF;
	border-radius:2px;
	background:#EDF6FF;
	margin-right:10px;
	color:#4D84FF;
	margin-top:10px;
	cursor:pointer;
}
.result-area{
	flex:1;
	display:flex;
	overflow:auto;
	flex-wrap:wrap;
}
.result-item{
	border:1px solid #E9E9E9;
	border-left:none;
	border-bottom:none;
	display:flex;
	flex-direction:column;
	min-height:300px;
	width:100%;
	min-width:500px;
}
.item-header{
	height:30px;
	font-size:14px;
	color:#333;
	font-weight:bold;
	margin:10px 0px 0px 20px;
}
.sizeTitle{
	display:inline-block;
	width:70px;
	height:38px;
	background:#f5f7fa;
	border:1px solid #dcdfe6;
	line-height:40px;
	text-align:center;
	color:#909399;
	border-left:none;
	margin-left:-3px;
	vertical-align:bottom;
	border-right:none;
}
.sizeCls .el-input{
	width:90px !important;
}
.sizeCls .el-input .el-input__inner{
	height:40px;
	width:90px !important;
	font-weight:normal !important;
	font-size:12px !important;
	margin-left:-3px;
}

.pingCls .el-input{
	width:60px !important;
}
.pingCls .el-input .el-input__inner{
	height:40px;
	width:60px !important;
	font-weight:normal !important;
	font-size:12px !important;
	margin-left:-3px;
}

.startButton{
	height:40px;
	border-radius:0px 4px 4px 0px;
	margin-left:-6px;
}
.item-title{
	height:70px;
	display:inline-block;
	width:50%;
	margin-right:-3px;
	line-height:40px;
	text-align:center;
	vertical-align:bottom;
	overflow:hidden;
	white-space:nowrap;
	text-overflow:ellipsis;
}
.item-title:hover::after{
	content:attr(placeholder);
	position: absolute;
	top: 10px;
	left:0px;
	display: block;
	word-break: keep-all;
	font-size: 12px;
	text-align: center;
	z-index:10000;
}
.item-title:before{
	content:attr(label);
	font-size:12px;
	color:#999;
	height:30px;
	line-height:30px;
	display:block;
	background:#F6F7FB;
	text-align:center;
	font-weight:bold;
}
.item-body{
	flex:1;
	display:flex;
	flex-wrap:wrap;
}
.ping-title{
	width:30%;
	height:70px;
	border-right:1px solid #E9E9E9;
	text-align:center;
}
.ping-title:before{
	content:attr(label);
	display:block;
	color:#999;
	font-weight:bold;
	margin:15px 0px 10px 0px;
}
.trace-title{
	display:inline-block;
	margin-right:80px;
	height:30px;
}
.trace-title:before{
	content:attr(label);
	color:#999;
	font-weight:bold;
	margin:0 20px;
}
.testStyle{
	border-top:2px solid #4D84FF;
}
.testStyle .item-header{
	color:#4D84FF
}
.oper-area .el-icon-close:before{
	content:'\e778';
}
.snTitle{
	 display:inline-block;
	 font-size:12px;
	 color:#999;
	 font-weight:400;
	 margin-left:15px;
}
.readonly-cls .tree-op-span {
	position: relative;
	opacity: 0.5;
}
.readonly-cls .tree-op-span::before {
	position: absolute;
	content: '';
	display: inline-block;
	top: 0;
	right: 0;
	bottom: 0;
	left: 0;
	z-index: 1000;
}
.oper-area .el-input__inner{
	height:40px;
}
.urlHead{
	display:inline-block;
	height:38px;
	line-height:38px;
	width:30px;
	background:#F5F7FA;
	color:#909399;
	vertical-align:bottom;
	border:1px solid #dcdfe6;
	border-radius:4px 0px 0px 4px;
	padding:0 20px;
	margin-right:-3px;
}
</style>
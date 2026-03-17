<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#statisticEventLog{
	height: 100%;
	padding:0 15px;
	position:relative;
	background:#FFFFFF;
}
.enb-el-slide-top{
	top:0;
	left:0;
	right:0;
	bottom:0;
}
#statisticEventLog .simpleQuery{
	padding: 10px 0px;
}
#statisticEventLog .simpleQuery .el-form-item{
	margin-bottom:0;
}

#statisticEventLog .circleIcon {
	top:7px;
}

.enbChartStyle{
	height:300px;
	margin-bottom:40px;
	border:1px solid #d1ecf5;
}
#eventLogCharts:empty::before {
	content: '<%=rb.getString("MeiShuJu")%>';
	display: inline-block;
	position: absolute;
	left: calc(50% - 30px);
	top: calc(50% - 10px);
}
</style>

<%-- 事件日志统计功能页面 --%>
<div id="statisticEventLog" class="flex-ctn">
	<!-- 按钮  -- 生成图表 -->
	
	<div class="circleIcon placeholder-bt" style="right: 95px;" placeholder="<%=rb.getString("TuBiao")%>">		
		<span class="el-icon el-icon-circle-chart" @click="createChart"></span>
	</div>
	
	<!-- 按钮  -- 导出 -->
	
	<div class="circleIcon placeholder-bt" style="right: 55px;" placeholder="<%=rb.getString("DaoChu")%>">		
		<span class="el-icon el-icon-circle-export" @click="exportStatisticData"></span>
	</div>
	
	
	<!-- 按钮  -- 返回上一功能页面 -->
	<div class="circleIcon">
		<span class="el-icon el-icon-circle-goback" @click="goBackPage"></span>
		<div class="titleButtonText"><%=rb.getString("FanHui")%></div>
	</div>
	
	<!-- 按钮  -- 返回列表 -->
	<div class="circleIcon" :style='styleObj'>
		<span class="el-icon el-icon-circle-table" @click="changeToTable"></span>
		<div class="titleButtonText"><%=rb.getString("LieBiao")%></div>
	</div>

	<el-ctable id="statisticEventLogLists" ref="ctable" height="100%" :rownumber=true :url="tbURL"
		pagination="true" :query-params="params_statistic">
		
		<!-- 查询 -- 事件日志统计 -->
		<template slot="toolbar">
			<el-form :model='params_statistic' ref="params_statistic" inline="true" class="simpleQuery">
				<el-form-item>
					<el-input v-model="params_statistic.id" placeholder="ID"></el-input>
				</el-form-item>
				<el-form-item>
					<el-select v-model="params_statistic.eventName" placeholder='<%=rb.getString("QingXuanZe")%>'>
						<el-option v-for="item in statisticOptions" :key="item.id" :label="item.text" :value="item.id">
						</el-option>
					</el-select>
				</el-form-item>
				<el-form-item style="margin-right:15px;">
					<el-date-picker type="datetimerange" v-model="dateValueStas" start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
					                value-format="yyyy-MM-dd HH:mm:ss" end-placeholder='<%=rb.getString("JieShuShiJian")%>' ></el-date-picker>
				</el-form-item>
				<i @click='searchResult' class="el-icon el-icon-common-search" style="vertical-align:top;"></i>
			</el-form>	
		</template>
		
		<!-- 主列表 -->
		<el-table-column label='ID' width="60" prop="id"></el-table-column>
		<el-table-column label='<%=rb.getString("Title_SheBeiBianMa")%>' min-width="200" prop="ne_code"></el-table-column>
		<el-table-column label='<%=rb.getString("CaoZuoIPAddress")%>' min-width="150"  prop="ipAddress" ></el-table-column>
		<el-table-column label='<%=rb.getString("RiZhiMingCheng")%>' min-width="100" prop="eventName" ></el-table-column>
		<el-table-column label='<%=rb.getString("TongJiCiShu")%>' min-width="150" prop="count"></el-table-column>
	</el-ctable>
		
	<!-- 二级页面 -- 查看生成的图表 -->
	<el-slide class="enb-el-slide-top" ref="slide" :footer="false" :header='false' position="top" :modal='modal'>
		<div id="eventLogCharts" style="height:100%;overflow:auto;"></div>
	 </el-slide>
	 	
	<form id = "exportStatistic" style="display:none" method="post"></form>
</div>

<script type="text/javascript">
new Vue({
	el:'#statisticEventLog',
	data(){
		var vm = this;
		
		return {
			showTable:true,
			addText:false,
			statisticOptions:[],
			dateValueStas:[],
			params_statistic:{
				timeZone:timeZone,
				id:'',
				eventName:'',
				startTime:'',
				endTime:''
			},
		  	styleObj:{
		    	zIndex:-10
		    },
			rowData : [],
			tbURL: ''
		}
	},
	props:['params_event'],
	methods:{ 
		init:function(params){
			var vm = this;
			vm.params_statistic.id = params.id;
			vm.params_statistic.eventName = params.eventName
			vm.params_statistic.startTime = params.startTime
			vm.params_statistic.endTime = params.endTime
			vm.tbURL = '${ctx}/cell/cellEventLog/getNumOfRebootList.action';
			
			//获取事件日志类型 -- 填充下拉选项 
			axios.post('${ctx}/cell/cellEventLog/getEventTypeForCombobox.action',stringify({
				
			})).then(function(response){
				let data = response.data
				vm.statisticOptions = data;
			}).catch(function(error){})
		},
		query(val){ // 模糊查询
			this.resetQuery();
			this.queryForm.search_text = val;
			this.$refs.cpairgrid.reload();
		},
		createChart(){ // 生成图表
	    	var vm = this;
	    	var message = '';
	    	vm.$root.styleObj = {
	    			zIndex:2010
	    	}
		    
	    	vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
				
    	    	//获取事件日志类型 -- 填充下拉选项 
    			axios.post('${ctx}/cell/cellEventLog/getStatisticsData.action',stringify({
    				timeZone : timeZone,
       				id :vm.params_statistic.id,
       				eventName:vm.params_statistic.eventName,
       				startTime:vm.params_statistic.startTime,
       				endTime:vm.params_statistic.endTime,
    			})).then(function(response){
    				let data = response.data
    			
    				
    				
    				var arr = Object.keys(data);
    				var type;
   					for(var i=0;i<arr.length;i++){
   						var arr1=[],arr2=[];
    					    var slength = data[arr[i]].length;
    					for(var j=0;j<slength;j++){
   							var res = data[arr[i]][j]
   							arr1.push(data[arr[i]][j].ne_code)
   							arr2.push(data[arr[i]][j].count)
   							type=arr[i]
   							
   						}
   						
   						vm.initChart(type,arr1,arr2);
   					}
   					
    			}).catch(function(error){})
    	    	
    	    	
    	    });
			
	    	
		},
		/**
		 * earchart图表生成处
		 * @param type:类型
		 * @param arr1:x轴数据
		 * @param arr2:y轴数据
		*/
		initChart(type,arr1,arr2){ 
			if(!(arr1.length && arr2.length)) {
				return;
			}
			var id = 'event'+type;
			var chartDiv = $("<div id='"+ id+"' class='enbChartStyle'></div>");
			var parDiv = $("#eventLogCharts");
			parDiv.append(chartDiv)
			
			var statisChart = echarts.init(document.getElementById(id));
	    	
	    	var option = {
	    			title:{
	    				text:type
	    			},
	    			
			         color : ['#85b1de','#e9a4a4'],
	    			 tooltip : {
			              trigger : 'axis',
			              backgroundColor : 'rgba(205,224,232,0.9)',
			              textStyle:{
			              	color:"#21608a",
			              	fontSize:12
			              }
	    			 },
	    			xAxis:[
	    				{
	    					type:'category',
	    					data:arr1,
	    					axisLine:{
	 				        	show : true,
	 			        		lineStyle:{
	 			        			color:"#b6b6b6"
	 			        		}
	 			        	},
	 	                    axisLabel : {
	 	                    	show:true,
	 	                    	rotate: 20,
	 	                    	interval: 0,
	 				        	textStyle:{                    		
	 		                    	color:"#b6b6b6"
	 	                    	},
	 	                    	lineStyle:{
	 	                    		color:"#b6b6b6"
	 	                    	},
	 	                    },
	 	                    axisTick: {
	 	                    	show: true,
	 	                    	interval: 0,
	 	                    	alignWithLabel: true,
	 	                    	
	 	                    }
	    				}
	    			],
	    			yAxis:[
	    				{
	    					type:'value',
    					 	axisLabel : {
					        	show:true,
					        	textStyle:{                    		
			                    	color:"#b6b6b6"
		                    	},
		                    	lineStyle:{
		                    		color:"#b6b6b6"
		                    	}
					        },
					        axisLine:{
				        		lineStyle:{
				        			color:"#b6b6b6"
				        		}
				        	},
				        	splitLine : {
				        		lineStyle:{
		                    		color:"#f1f1f4"
		                    	}
				        	}
	    				}
	    			],
	    			series:[
	    				{
	    					name:'Count',
	    					type:'bar',
	    					data:arr2,
	    					barWidth:30
	    				}
	    			]
	    	}
	    	statisChart.setOption(option);
		},
		searchResult(){ // 日志列表查询
			
			var vm = this;
			if(vm.dateValueStas != null){
				vm.params_statistic.startTime = vm.dateValueStas[0];
    			vm.params_statistic.endTime = vm.dateValueStas[1];
			}
			vm.$refs.ctable.refresh()
		},
		exportStatisticData(){ // 导出
			
			var vm = this;    		
    		/* $("#exportStatistic").form('submit', {
       	        url: '${ctx}/cell/cellEventLog/exportStatisticsDataToCsvFile.action',
       	        onSubmit: function(params){
       	        	params.timeZone = timeZone;
       				params.id = vm.params_statistic.id ;
       				params.eventName = vm.params_statistic.eventName;
       				params.startTime = vm.params_statistic.startTime;
       				params.endTime = vm.params_statistic.endTime;
       	        }
       	    });  */ 
       		exportByForm('${ctx}/cell/cellEventLog/exportStatisticsDataToCsvFile.action',{
       			timeZone: timeZone,
       			id: vm.params_statistic.id,
       			eventName: vm.params_statistic.eventName,
       			startTime: vm.params_statistic.startTime,
       			endTime: vm.params_statistic.endTime
       		});
		},
		goBackPage(){ // 返回上一页
			var vm = this;
			vm.$root.styleObj = {
	    			zIndex:-10
	    	};

			eventBus.$emit('hide-collect')
		},
		changeToTable(){ // 返回列表
			var vm = this;
			vm.$root.styleObj = {
	    			zIndex:-10
	    	};
		    vm.$refs.slide.hide();
		    $("#eventLogCharts").children().remove();
		},
		 showText(){
	    	this.addText = true;
	    },
	    hideText(){
	    	this.addText = false;
	    }
	},
	mounted(){
		eventBus.$off('statisticLog').$on('statisticLog',this.init);
	}
})
</script>
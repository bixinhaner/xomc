<%@ page import="java.util.Locale"%>
<%@ include file="/common/taglibs.jsp"%>
<%@ page contentType="text/html;charset=UTF-8"%>
<style>
    #gnbSettingStatisticsPages{
        width: calc(100% - 2px); 
        height: 100%;
    }
    #gnbSettingStatisticsPages .gnbSettingStatisticsMainBoxCls{
        display: flex; 
        flex-direction: column;  
        width: 100%; 
        height: 100%;
        gap: 10px;
    }
    #gnbSettingStatisticsPages .gnbSettingStatisticsItemBoxCls {
        flex: 1;
        min-height: 325px;
        padding: 20px;
        background: #FFFFFF;
        box-sizing: border-box;
        border: 1px solid #D5DCEC;
        border-radius: 8px;
        overflow: hidden;
    }
    #gnbSettingStatisticsPages .gnbSettingStatisticsBox {
        position: relative;
        display: flex;
        width: 100%;
        height: calc(100% - 30px);
    }
    #gnbSettingStatisticsPages .gnbSettingStatisticsCls{
        width:100%;
        height:100%;
        padding-bottom:20px; 
        border: 0;
    }
</style>
<div id="gnbSettingStatisticsPages">
	<div class="gnbSettingStatisticsMainBoxCls">
		<div class='gnbSettingStatisticsItemBoxCls'>
			<div class='commonText14' >gNB <%=rb.getString("ZaiXianZhuangTai")%></div>
			<div class="gnbSettingStatisticsBox">
				<div class="gnbSettingStatisticsCls" id="echart_gnb_online"></div>
			</div>
		</div>
		<div class='gnbSettingStatisticsItemBoxCls'>
			<div class='commonText14' >gNB <%=rb.getString("ShiFouJiHuo")%></div>
			<div  class="gnbSettingStatisticsBox">
				<div class="gnbSettingStatisticsCls" id="echart_gnb_active" ></div>
			</div>
		</div>
	
	</div>
</div>

<script type="text/javascript">
var timeParam = getNowTimeToZoneTimeRange(timeZone, 144),
	start_time_gnb = timeParam.start_time.substring(0,11)+"00:00:00",
	end_time_gnb = timeParam.end_time,
	smallCellCode = gnbTabSettingVue.rowData.small_cell_code,
	pointerCount = 6*24+1;// 24*(60/5);
	
	var gnbSettingStatisticsPages = new Vue({
	    el: '#gnbSettingStatisticsPages',
	    data() {
	    	return {
				tableData: []
	    	}
	    },
	    methods: {
	    	statisticsInit(){
				var vm = this, small_cell_code = gnbTabSettingVue.rowData.small_cell_code,
					param ={
						isGnb: 1,
						small_cell_code: small_cell_code
					};
				
				loadAjax('echart_gnb_online','offline',function(){
					setTimeout(function(){
						var dom = document.querySelector('#echart_gnb_online');
						if(dom) {
							var chart = echarts.getInstanceByDom(dom);
							// 初始化时间轴切换事件
							chart.off('timelinechanged');
						}
					},0)
				});

				loadAjax('echart_gnb_active','active',function(){
					setTimeout(function(){
						var dom = document.querySelector('#echart_gnb_active');
						if(dom) {
							var chart = echarts.getInstanceByDom(dom);
							// 初始化时间轴切换事件
							chart.off('timelinechanged');
						}
					},0)
				});
                // 窗口缩放时自适应
                window.removeEventListener('resize',vm.resizeChart);
                window.addEventListener('resize',vm.resizeChart);
			},
			// 图表自适应
            resizeChart(){
                var list = ['echart_gnb_online', 'echart_gnb_active'];
                
                list.map(function(id){
                    var dom = document.querySelector('#'+id);
                    if(dom) {
                        var itn = echarts.getInstanceByDom(dom);
                        itn && itn.resize();
                    }
                })
            },
	    },
		mounted(){
	    	this.statisticsInit();
	    }
	});
	/**
	* 请求后台统计图表数据
	* @param idName{number}：图表容器Id
	* @param type{number}：类型
	* @param fn{number}：回调方法
	**/
	function loadAjax(idName,type,fn){
		dataArr=[];
		var params = {
				isGnb: 1,
				device_type: "enb",
				device_code: smallCellCode,
				time_level: "min",
				timeZone: timeZone,
				start_time: start_time_gnb,
				end_time: end_time_gnb
	 	}
		// 获取统计数据
		$.post("${ctx}/system/device/getDeviceStatusDataList.action", params, function(data){
			if(data){
			}else{
				data = [];
			}
			dataArr=[];
			$.each(data,function(index,item){
				var xDate=item.statistics_time.split(' ');
				var activeSta=item.active_status;
				var onlineSta=item.online_status;
				var ueSta=item.ue_count;
				if(activeSta == 0){
					activeSta = activeSta+1;
				}else if(activeSta == 1){
					activeSta = activeSta+2
				}
				if(onlineSta == 0){
					onlineSta = onlineSta+1;
				}else if(onlineSta == 1){
					onlineSta = onlineSta+2
				}
				dataArr[index]=[xDate[0],xDate[1],activeSta,onlineSta,ueSta];
			});
			// 初始化系列数据
			finalArr = [];
			for(var initIndex=0;initIndex<7;initIndex++ ){
				var arrIndex = getYesterDay(initIndex);
				if(!finalArr[arrIndex]){
					var timeaxisArr = [], activeaxisArr = [],offlineaxisArr=[],ueaxisArr=[];
					for(var axisIndex=0; axisIndex<pointerCount; axisIndex++){
						var yaxisStartTime = getYesterDay(initIndex).substring(0,10)+' 00:00:00',
							offsetTime = addTimes(new Date(yaxisStartTime),axisIndex*10);
						timeaxisArr.push(formatDate(offsetTime));
						if(offsetTime.getTime() > new Date(end_time_gnb).getTime()){
							activeaxisArr.push(undefined);
							offlineaxisArr.push(undefined);
							ueaxisArr.push(undefined);
						}else{
							activeaxisArr.push(1);
							offlineaxisArr.push(1);
							ueaxisArr.push(0);
						}
					}
					finalArr[arrIndex] = [];
					finalArr[arrIndex]['time'] = timeaxisArr;
					finalArr[arrIndex]['active'] = activeaxisArr;
					finalArr[arrIndex]['offline'] = offlineaxisArr;
					finalArr[arrIndex]['ue'] = ueaxisArr;
				}
			}
			// 初始化系列的x、y轴数据
			$.each(dataArr,function(n,m){
				var dateIndex = m[0],timeValue=m[1].substring(0,5),activeValue=m[2],offlineValue=m[3],ueValue=m[4];
				if(finalArr[dateIndex]['time'].includes(dateIndex+' '+m[1])){
					var differTimes = new Date(dateIndex+' '+m[1]).getTime() - new Date(dateIndex+' 00:00:00').getTime();
					var axisDateIndex = Math.round(differTimes/(1000*60*10));
					finalArr[dateIndex]['active'].splice(axisDateIndex,1,activeValue);
					finalArr[dateIndex]['offline'].splice(axisDateIndex,1,offlineValue); 
					finalArr[dateIndex]['ue'].splice(axisDateIndex,1,ueValue); 
				}
			});

			var preData = null;
			for(var key in finalArr){
				if(typeof finalArr[key] != 'function'){
					if(preData){
						finalArr[key]['active'][pointerCount-1] = preData['active'][0];
						finalArr[key]['offline'][pointerCount-1] = preData['offline'][0];
						finalArr[key]['ue'][pointerCount-1] = preData['ue'][0];
						preData = finalArr[key]; 
					}else preData = finalArr[key];
				}
			}
			if(fn) try{fn();}catch(e){}
			createEchart(finalArr,idName,type);
		}, "json");
	}
	/**
	* 绘制统计页面图表
	* @param finalArr{array}：初始化后的图表系列数据
	* @param idName{string}：图表容器Id
	* @param type{string}：类型
	**/
	function createEchart(finalArr,idName,type){
		var preValue;
		$("#"+idName).css("display","block");
		var enodebChart = echarts.init(document.getElementById(idName));
	 	 var option = {
				baseOption:{
					 timeline: {
			              axisType: 'category',  
			              show: true,  
			              autoPlay: false,  
			              playInterval: 1000,  
			              data: [getYesterDay(6).substring(5).replace("-","."),
			                     getYesterDay(5).substring(5).replace("-","."),
			                     getYesterDay(4).substring(5).replace("-","."),
			                     getYesterDay(3).substring(5).replace("-","."),
			                     getYesterDay(2).substring(5).replace("-","."),
			                     getYesterDay(1).substring(5).replace("-","."),
			                     getYesterDay(0).substring(5).replace("-",".")],
			              notMerge:true,
			              controlPosition:'none',
			              currentIndex:6,
			              checkpointStyle:{
			            	  color:'#209FFF',
			            	  borderColor:'none'
			              },
						  lineStyle : { color : '#B0AFBA',width : 1 },
						  itemStyle : {
						  	  normal : { borderColor : '#B0AFBA' },
							  emphasis : {
								  borderColor : '#1e90ff',
								  color : '#1e90ff'
							  }
						  },
			              label:{
			            	  emphasis:{
			            		  color:'#209FFF'
			            	  }
			              }
			          },
			           grid:{
			        	  top:50,
			        	  bottom:80
			          },
					  color: ['#47D468','#69B6FC'],
			          tooltip:{
			        	  trigger:"axis",
			              padding:[10,5,10,5],
			              formatter:function(params){
			            	  if(params.length){
			            		  var ueStr = "";
				            	  var str = "";
				            	   if(idName == "echart_enb_ue"){
				            		  ueStr += "<div>" + params[0].name + "</div>";
				            		  if(params[0].data == undefined){
				            			  ueStr += "<div>UE:-</div>" 
				            		  }else{
				            			  ueStr += "<div>UE:" + params[0].data + "</div>"
				            		  }
				            		  return ueStr;
				            	  }else{
				            		  str += "<div>" + params[0].name + "</div>";
				            		  return str;
				            	  }
			            	  }
			              }
			          },
			          xAxis: [{ 
		            	   name:'<%=rb.getString("XiaoShi") %>',
				    	   type:"category",
				    	   axisLabel:{
					    		  show:true,
					    		  formatter : function(val) {
					                	var secondTime = val.split(' ')[1];
					            		clock = secondTime.substring(0,2);
					                 if(secondTime.substring(3,5)=='00') return clock;
					                 return val;
					              },
		                          interval:function(index){
		                        	  if(index%6 == 0 && index !=pointerCount-1){
		                        		  return true;
		                        	  }
		                          }
					    	 },
							axisLine:{
								lineStyle:{ color:"#666666" }
							},
							splitLine:false ,
							boundaryGap:false
		              }],
		              yAxis: { 
		            	    minInterval: (idName=='echart_enb_ue')?1:1,
		            	    max: (idName=='echart_enb_ue')?null:4,
		            	    name: (idName=='echart_enb_ue')?'<%=rb.getString("GeShuUe") %>':'<%=rb.getString("ZhuangTai") %>', 
							type:"value",
							axisLine:{
								show:true,
								lineStyle:{ color:"#666666" },
								interval:12
							},
							axisTick:{
								show:false
							},
							splitLine:{
								show:idName =='echart_enb_ue'?true:false
							},
							axisLabel:{
								show:true,
								formatter:function(value,index){
									if(idName == 'echart_gnb_active'){
										var texts=[];
										if(value == 3){
											texts.push('<%=rb.getString("ShouYe_HuoYue")%>');
										}else if(value == 1){
											texts.push('<%=rb.getString("ShouYe_BuHuoYue")%>');
										}
										return texts;
									}else if(idName == 'echart_gnb_online'){
										var texts=[];
										if(value == 3){
											texts.push('<%=rb.getString("ShouYe_ZaiXian")%>');
										}else if(value == 1){
											texts.push('<%=rb.getString("ShouYe_BuZaiXian")%>');
										}
										return texts;
									}else if(idName == 'echart_enb_ue'){
										return value;
									}
								}
							}
		              },
		              series: [  
				                  {  
				                		type:'line',
				                		step:idName=='echart_enb_ue'?false:true,
				    		        	 markLine: idName=='echart_enb_ue'?{}:{
					                	  	data:[{
					                	  		name:' ',
					                	  		yAxis: 1
					                	  	},
					                	  	{
					                	  		name:' ',
					                	  		yAxis: 3
					                	  	}],
					                	  	animation: false,
					                	  	lineStyle:{
					                	  		normal:{
					                	  			type:'dotted',
					                	  			color:'#ccc',
					                	  			width:0.5,
					                	  			opacity:0.8
					                	  		}
					                	  	},
					                	  	symbolSize:0,
					                	  	silent:true,
					                	  	label:{
					                	  		normal:{
					                	  			show:false
					                	  		}
					                	  	}
					                  },
					                  itemStyle: {
											normal: {
												areaStyle: {opacity: 0.1}
											}
									  },
					                  symbolSize: 0
				                  }
				              ]  
				},	
				//变量则写在options中  
				options:[  
					{  
						xAxis: [{ 
							data:finalArr[getYesterDay(6)]['time']
						}],
						series: [
							{  
							data:finalArr[getYesterDay(6)][type]
							} 
						]  
					},  
					{  
						xAxis: [{ 
							data:finalArr[getYesterDay(5)]['time']
						}],
						series: [ 
							{  
							data:finalArr[getYesterDay(5)][type]
							} 
						]  
					},
					{  
						xAxis: [{  
							data:finalArr[getYesterDay(4)]['time']
						}], 
						series: [  
							{  
							data:finalArr[getYesterDay(4)][type]
							} 
						]  
					},
					{  
						xAxis: [{  
							data:finalArr[getYesterDay(3)]['time'],
							
						}],
						series: [
							{    
							data:finalArr[getYesterDay(3)][type]
							} 
						]  
					},
					{  
						
						xAxis: [{  
							data:finalArr[getYesterDay(2)]['time']
						}],
						series: [  
							{  
							data:finalArr[getYesterDay(2)][type]
							} 
						]  
					},
					{  
						xAxis: [{  
							data:finalArr[getYesterDay(1)]['time']
						}], 
						series: [ 
							{   
							data:finalArr[getYesterDay(1)][type]
							}  
						]  
					},
					{  
						
						xAxis: [{  
							data: finalArr[getYesterDay(0)]['time']
						}],
						series: [
							{    
							data:finalArr[getYesterDay(0)][type]
							} 
						]  
					}
				]
			  } 
		 enodebChart.setOption(option);
	 	 enodebChart.resize();
	}
</script> 